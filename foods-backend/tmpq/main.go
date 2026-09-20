package main
import("context";"fmt";"os";"github.com/jackc/pgx/v5/pgxpool";"github.com/foods-platform/foods-backend/internal/platform/config")
func main(){config.LoadDotEnv(".env");db,_:=pgxpool.New(context.Background(),os.Getenv("DATABASE_URL"));defer db.Close()
 rows,_:=db.Query(context.Background(),`SELECT code,channel,COALESCE(customer_name,''),status FROM orders ORDER BY created_at DESC LIMIT 8`)
 for rows.Next(){var c,ch,n,s string;rows.Scan(&c,&ch,&n,&s);fmt.Printf("%-14s %-10s %-15s %s\n",c,ch,fmt.Sprintf("%q",n),s)}
 rows.Close()}
