package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InitDB khởi tạo kết nối Database
func InitDB() *pgxpool.Pool {
	// Định dạng: postgres://user:password@localhost:5432/dbname
	// Hãy thay đổi thông số này theo máy của bạn
	connstring := os.Getenv("DB_URL")
	connStr := connstring
	fmt.Print(connStr)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Không thể cấu hình database: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi tạo connection pool: %v\n", err)
		os.Exit(1)
	}

	// Kiểm tra kết nối thật sự bằng cách Ping
	err = pool.Ping(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Không thể kết nối tới Postgres: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Kết nối PostgreSQL thành công!")
	return pool
}
