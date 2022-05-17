package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/otofuto/textreal/pkg/database"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("command line arguments not enough")
		return
	}
	if os.Args[1] == "panic" {
		panic("Panic!")
	}
	godotenv.Load()
	db := database.Connect()
	defer db.Close()

	sql := "insert into log (msg) values (?)"
	ins, err := db.Prepare(sql)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = ins.Exec(os.Args[1])
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("insert log (" + os.Args[1] + ")")
}
