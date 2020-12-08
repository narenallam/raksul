package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
)

func xmain() {
	word := flag.String("word", "", "a word")
	flag.Parse()

	r, err := zip.OpenReader("foc-slack-export.zip")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			if strings.Index(f.Name, ".json") == len(f.Name)-5 {
				rc, err := f.Open()
				if err != nil {
					panic(err)
				}

				b, err := ioutil.ReadAll(rc)
				if err != nil {
					panic(err)
				}
				rc.Close()

				var all []map[string]interface{}
				err = json.Unmarshal(b, &all)
				if err != nil {
					panic(err)
				}

				for _, m := range all {

					if text, ok := m["text"].(string); ok {

						words := strings.Split(text, " ")

						for i, w := range words {
							if w == *word {
								before := make([]string, 4)
								if i-4 >= 0 {
									before[0], before[1], before[2], before[3] = words[i-4], words[i-3], words[i-2], words[i-1]
								}

								after := make([]string, 4)
								if i+4 < len(words) {
									after[0], after[1], after[2], after[3] = words[i+1], words[i+2], words[i+3], words[i+4]
								}

								fmt.Printf("File %s : %s %s %s %s %s %s %s %s %s\n", f.Name, before[0], before[1], before[2], before[3], w, after[0], after[1], after[2], after[3])
							}
						}
					}
				}
			}
		}
	}
}
