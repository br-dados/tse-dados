// Command migrate aplica as migrations embutidas das tabelas tse_*.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/danyele/tse-dados/internal/database"
)

func main() {
	var host, port, user, pass, db string
	var maxConns int
	flag.StringVar(&host, "pg-host", "127.0.0.1", "host")
	flag.StringVar(&port, "pg-port", "5432", "porta")
	flag.StringVar(&user, "pg-user", "postgres", "usuario")
	flag.StringVar(&pass, "pg-password", "", "senha")
	flag.StringVar(&db, "pg-db", "tsedados", "banco")
	flag.IntVar(&maxConns, "pg-max-conns", 10, "maximo de conexoes")
	flag.Parse()

	if pass == "" {
		fatal("--pg-password e obrigatoria")
	}
	ctx := context.Background()
	pool, err := database.NewPool(ctx, database.Config{
		Host: host, Port: port, User: user, Password: pass, Database: db,
		MaxConns: int32(maxConns), MinConns: 2,
	})
	if err != nil {
		fatal(err.Error())
	}
	defer pool.Close()

	err = database.Migrate(ctx, pool)
	if err != nil && err != database.ErrNoPendingMigrations {
		fatal("migrate: " + err.Error())
	}
	fmt.Println("migracoes aplicadas (ou nenhuma pendente)")
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "erro:", msg)
	os.Exit(1)
}
