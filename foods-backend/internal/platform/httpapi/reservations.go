package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type reservationView struct {
	ID            string  `json:"id"`
	CustomerID    *string `json:"customerId,omitempty"`
	CustomerName  string  `json:"customerName"`
	CustomerPhone string  `json:"customerPhone"`
	StartsAt      string  `json:"startsAt"`
	Guests        int     `json:"guests"`
	TableID       *string `json:"tableId,omitempty"`
	TableName     string  `json:"tableName"`
	Status        string  `json:"status"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"createdAt"`
}

type reservationInput struct {
	CustomerID    *string `json:"customerId,omitempty"`
	CustomerName  string  `json:"customerName"`
	CustomerPhone string  `json:"customerPhone"`
	StartsAt      string  `json:"startsAt"`
	Guests        int     `json:"guests"`
	TableID       *string `json:"tableId,omitempty"`
	Notes         string  `json:"notes"`
}

func validReservationStatus(status string) bool {
	return status=="pending"||status=="confirmed"||status=="seated"||status=="cancelled"||status=="no_show"
}

func validReservationTransition(current,next string) bool {
	if current==next{return true}
	switch current {
	case "pending":
		return next=="confirmed"||next=="seated"||next=="cancelled"||next=="no_show"
	case "confirmed":
		return next=="seated"||next=="cancelled"||next=="no_show"
	default:
		return false
	}
}

func normalizeOptionalID(value *string) *string {
	if value==nil{return nil}
	trimmed:=strings.TrimSpace(*value)
	if trimmed==""{return nil}
	return &trimmed
}

func parseReservationInput(in *reservationInput)(time.Time,bool){
	in.CustomerID=normalizeOptionalID(in.CustomerID)
	in.TableID=normalizeOptionalID(in.TableID)
	in.CustomerName=strings.TrimSpace(in.CustomerName)
	in.CustomerPhone=strings.TrimSpace(in.CustomerPhone)
	in.Notes=strings.TrimSpace(in.Notes)
	startsAt,err:=time.Parse(time.RFC3339,in.StartsAt)
	if err!=nil||startsAt.Before(time.Now().Add(-5*time.Minute))||in.CustomerName==""||len(in.CustomerName)>180||len(in.CustomerPhone)>40||in.Guests<1||in.Guests>100||len(in.Notes)>500{
		return time.Time{},false
	}
	return startsAt,true
}

func scanReservation(row pgx.Row)(reservationView,error){
	var item reservationView
	err:=row.Scan(&item.ID,&item.CustomerID,&item.CustomerName,&item.CustomerPhone,&item.StartsAt,&item.Guests,&item.TableID,&item.TableName,&item.Status,&item.Notes,&item.CreatedAt)
	return item,err
}

func reservationColumns() string {
	return `r.id::text,r.customer_id::text,r.customer_name,r.customer_phone,
	to_char(r.starts_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),r.guests,r.table_id::text,
	COALESCE((SELECT name FROM tables rt WHERE rt.id=r.table_id AND rt.organization_id=r.organization_id AND rt.location_id=r.location_id),''),
	r.status,r.notes,to_char(r.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')`
}

func (a *API) validateReservationReferences(r *http.Request, s scope, in reservationInput, excludeID string) *apiError {
	if in.CustomerID!=nil {
		var ok bool
		if err:=a.db.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1 AND organization_id=$2 AND active)`,*in.CustomerID,s.OrganizationID).Scan(&ok);err!=nil||!ok{
			return &apiError{Code:"invalid_customer",Message:"El cliente seleccionado no está disponible."}
		}
	}
	if in.TableID!=nil {
		var seats int
		err:=a.db.QueryRow(r.Context(),`SELECT seats FROM tables WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND active`,*in.TableID,s.OrganizationID,s.LocationID).Scan(&seats)
		if errors.Is(err,pgx.ErrNoRows){return &apiError{Code:"invalid_table",Message:"La mesa seleccionada no está disponible en este local."}}
		if err!=nil{return &apiError{Code:"reservation_unavailable",Message:"No pudimos validar la mesa."}}
		if in.Guests>seats{return &apiError{Code:"table_capacity",Message:"La mesa seleccionada no tiene capacidad suficiente."}}
		var conflict bool
		if err=a.db.QueryRow(r.Context(),`
			SELECT EXISTS(
			 SELECT 1 FROM reservations
			 WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
			   AND status IN ('pending','confirmed','seated')
			   AND starts_at=$4
			   AND ($5='' OR id::text<>$5)
			)`,s.OrganizationID,s.LocationID,*in.TableID,in.StartsAt,excludeID).Scan(&conflict);err!=nil{
			return &apiError{Code:"reservation_unavailable",Message:"No pudimos validar la disponibilidad de la mesa."}
		}
		if conflict{return &apiError{Code:"reservation_conflict",Message:"La mesa ya tiene una reserva en ese horario."}}
	}
	return nil
}

func (a *API) listReservations(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	page,size:=pageParams(r)
	q:=strings.TrimSpace(r.URL.Query().Get("q"))
	status:=strings.TrimSpace(r.URL.Query().Get("status"))
	if status!=""&&!validReservationStatus(status){fail(w,400,"invalid_reservation_status","El estado de reserva no es válido.");return}
	where:=`r.organization_id=$1 AND r.location_id=$2
		AND ($3='' OR r.customer_name ILIKE '%'||$3||'%' OR r.customer_phone ILIKE '%'||$3||'%' OR COALESCE(t.name,'') ILIKE '%'||$3||'%')
		AND ($4='' OR r.status=$4)`
	var total int
	if err:=a.db.QueryRow(r.Context(),`SELECT count(*) FROM reservations r LEFT JOIN tables t ON t.id=r.table_id AND t.organization_id=r.organization_id AND t.location_id=r.location_id WHERE `+where,s.OrganizationID,s.LocationID,q,status).Scan(&total);err!=nil{
		fail(w,503,"reservations_unavailable","No pudimos cargar las reservas.");return
	}
	rows,err:=a.db.Query(r.Context(),`
		SELECT `+reservationColumns()+`
		FROM reservations r
		LEFT JOIN tables t ON t.id=r.table_id AND t.organization_id=r.organization_id AND t.location_id=r.location_id
		WHERE `+where+`
		ORDER BY CASE WHEN r.starts_at>=now() THEN 0 ELSE 1 END,r.starts_at ASC,r.created_at DESC
		LIMIT $5 OFFSET $6`,s.OrganizationID,s.LocationID,q,status,size,(page-1)*size)
	if err!=nil{fail(w,503,"reservations_unavailable","No pudimos cargar las reservas.");return}
	defer rows.Close()
	items:=[]reservationView{}
	for rows.Next(){
		item,scanErr:=scanReservation(rows)
		if scanErr!=nil{fail(w,503,"reservations_unavailable","No pudimos leer las reservas.");return}
		items=append(items,item)
	}
	writeJSON(w,200,map[string]any{"items":items,"total":total,"page":page,"pageSize":size})
}

func (a *API) createReservation(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in reservationInput
	if json.NewDecoder(r.Body).Decode(&in)!=nil{fail(w,400,"invalid_reservation","Revisa los datos de la reserva.");return}
	startsAt,ok:=parseReservationInput(&in)
	if !ok{fail(w,400,"invalid_reservation","Completa cliente, fecha, hora y número de personas.");return}
	in.StartsAt=startsAt.UTC().Format(time.RFC3339)
	if problem:=a.validateReservationReferences(r,s,in,"");problem!=nil{fail(w,400,problem.Code,problem.Message);return}
	item,err:=scanReservation(a.db.QueryRow(r.Context(),`
		INSERT INTO reservations AS r(organization_id,location_id,customer_id,customer_name,customer_phone,starts_at,guests,table_id,notes,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING `+reservationColumns()+`
	`,s.OrganizationID,s.LocationID,in.CustomerID,in.CustomerName,in.CustomerPhone,startsAt,in.Guests,in.TableID,in.Notes,s.UserID))
	if err!=nil{fail(w,503,"reservations_unavailable","No pudimos registrar la reserva.");return}
	a.audit(r,"reservation.created","reservation",item.ID)
	writeJSON(w,201,item)
}

func (a *API) updateReservation(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	id:=r.PathValue("id")
	var in reservationInput
	if json.NewDecoder(r.Body).Decode(&in)!=nil{fail(w,400,"invalid_reservation","Revisa los datos de la reserva.");return}
	startsAt,ok:=parseReservationInput(&in)
	if !ok{fail(w,400,"invalid_reservation","Completa cliente, fecha, hora y número de personas.");return}
	in.StartsAt=startsAt.UTC().Format(time.RFC3339)
	if problem:=a.validateReservationReferences(r,s,in,id);problem!=nil{fail(w,400,problem.Code,problem.Message);return}
	var current string
	if err:=a.db.QueryRow(r.Context(),`SELECT status FROM reservations WHERE id=$1 AND organization_id=$2 AND location_id=$3`,id,s.OrganizationID,s.LocationID).Scan(&current);errors.Is(err,pgx.ErrNoRows){fail(w,404,"reservation_not_found","La reserva no existe.");return}else if err!=nil{fail(w,503,"reservations_unavailable","No pudimos validar la reserva.");return}
	if current!="pending"&&current!="confirmed"{fail(w,409,"reservation_not_editable","Solo una reserva pendiente o confirmada puede editarse.");return}
	item,err:=scanReservation(a.db.QueryRow(r.Context(),`
		UPDATE reservations r SET customer_id=$4,customer_name=$5,customer_phone=$6,starts_at=$7,guests=$8,table_id=$9,notes=$10,updated_at=now()
		WHERE r.id=$1 AND r.organization_id=$2 AND r.location_id=$3
		RETURNING `+reservationColumns()+`
	`,id,s.OrganizationID,s.LocationID,in.CustomerID,in.CustomerName,in.CustomerPhone,startsAt,in.Guests,in.TableID,in.Notes))
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"reservation_not_found","La reserva no existe.");return}
	if err!=nil{fail(w,503,"reservations_unavailable","No pudimos actualizar la reserva.");return}
	a.audit(r,"reservation.updated","reservation",item.ID)
	writeJSON(w,200,item)
}

func (a *API) updateReservationStatus(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in struct{Status string `json:"status"`}
	if json.NewDecoder(r.Body).Decode(&in)!=nil||!validReservationStatus(in.Status){fail(w,400,"invalid_reservation_status","El estado de reserva no es válido.");return}
	tx,err:=a.db.Begin(r.Context())
	if err!=nil{fail(w,503,"reservations_unavailable","No pudimos iniciar la actualización.");return}
	defer tx.Rollback(r.Context())
	var current string
	err=tx.QueryRow(r.Context(),`SELECT status FROM reservations WHERE id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`,r.PathValue("id"),s.OrganizationID,s.LocationID).Scan(&current)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"reservation_not_found","La reserva no existe.");return}
	if err!=nil{fail(w,503,"reservations_unavailable","No pudimos validar la reserva.");return}
	if !validReservationTransition(current,in.Status){fail(w,409,"invalid_reservation_transition","La reserva ya no admite ese cambio de estado.");return}
	if _,err=tx.Exec(r.Context(),`UPDATE reservations SET status=$4,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`,r.PathValue("id"),s.OrganizationID,s.LocationID,in.Status);err!=nil{fail(w,503,"reservations_unavailable","No pudimos actualizar la reserva.");return}
	if err=tx.Commit(r.Context());err!=nil{fail(w,503,"reservations_unavailable","No pudimos confirmar la actualización.");return}
	a.audit(r,"reservation.status_updated","reservation",r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}
