package main

import(
	"net/http"
	"os"
	"log"
)

func main(){
	a := os.Args
	dir := "."
	port := ":8080"
	if len(a) > 1{
		dir = a[1]
	}
	if len(a) > 2{
		port = a[2]
	}
	if err := http.ListenAndServe(port, http.FileServer(http.Dir(dir))); err != nil{
		log.Fatalln("httpd:",err)
	}
}
		