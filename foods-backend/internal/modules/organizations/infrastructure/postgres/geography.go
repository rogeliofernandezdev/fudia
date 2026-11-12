package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
)

func (r *Organizations) GeographicAreas(ctx context.Context, parent string) ([]domain.GeographicArea, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT g.code,COALESCE(g.parent_code,''),g.name,concat_ws(' · ',g.name,pr.name,dp.name)
   FROM geographic_areas g LEFT JOIN geographic_areas pr ON pr.code=g.parent_code
   LEFT JOIN geographic_areas dp ON dp.code=pr.parent_code
   WHERE COALESCE(g.parent_code,'')=$1 ORDER BY g.name,g.code`, parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.GeographicArea{}
	for rows.Next() {
		var item domain.GeographicArea
		if err = rows.Scan(&item.Code, &item.ParentCode, &item.Name, &item.Label); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
