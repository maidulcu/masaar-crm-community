// Command openapi converts swag's Swagger 2.0 output into docs/openapi.json (OpenAPI 3.1).
// Run scripts/openapi.sh rather than this directly.
//
//	go run ./cmd/openapi -in docs/swagger.json -out docs/openapi.json [-check]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/maidulcu/masaar-crm/internal/api"
)

func main() {
	in := flag.String("in", "docs/swagger.json", "Swagger 2.0 input (from swag)")
	out := flag.String("out", "docs/openapi.json", "OpenAPI 3.1 output")
	check := flag.Bool("check", false, "fail if -out is not up to date instead of writing it")
	verbose := flag.Bool("v", false, "list dropped and undocumented routes")
	flag.Parse()

	src, err := os.ReadFile(*in)
	if err != nil {
		log.Fatal(err)
	}
	doc, rep, err := api.BuildOpenAPI(src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("openapi: %d operations documented, %d annotations dropped (no such route), %d routes undocumented\n",
		rep.Documented, len(rep.Dropped), len(rep.Undocumented))
	if *verbose {
		for _, d := range rep.Dropped {
			fmt.Println("  dropped     ", d)
		}
		for _, u := range rep.Undocumented {
			fmt.Println("  undocumented", u)
		}
	}
	if *check {
		cur, _ := os.ReadFile(*out)
		if !bytes.Equal(cur, doc) {
			log.Fatalf("%s is out of date: run scripts/openapi.sh", *out)
		}
		return
	}
	if err := os.WriteFile(*out, doc, 0o644); err != nil {
		log.Fatal(err)
	}
}
