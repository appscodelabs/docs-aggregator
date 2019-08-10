package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	shell "github.com/codeskyblue/go-sh"
	"github.com/gohugoio/hugo/parser"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := dostuff(); err != nil {
		log.Fatalln(err)
	}
}

const dir = "/home/tamal/Desktop/docs"

func dostuff() error {
	data, err := ioutil.ReadFile("/home/tamal/Desktop/products.json")
	if err != nil {
		return err
	}

	var cfg DocAggregator
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}

	//dir, err := ioutil.TempDir(dir, "aggr")
	//if err != nil {
	//	return err
	//}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	sh := shell.NewSession()
	sh.SetDir(dir)
	sh.ShowCMD = true

	for name, p := range cfg.Products {
		p.Name = name
		err = processProduct(p, dir, sh)
		if err != nil {
			return err
		}
		break // skip
	}

	return nil
}

func processProduct(p Product, rootDir string, sh *shell.Session) error {
	prjDir := filepath.Join(rootDir, p.Name)
	err := os.MkdirAll(prjDir, 0755)
	if err != nil {
		return err
	}

	repoDir := filepath.Join(prjDir, "repo")
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	docsDir := filepath.Join(prjDir, "docs")
	err = os.MkdirAll(docsDir, 0755)
	if err != nil {
		return err
	}

	err = sh.Command("git", "clone", p.GithubURL, repoDir).Run()
	if err != nil {
		return err
	}

	for _, v := range p.Versions {
		if !v.HostDocs {
			continue
		}

		if v.DocsDir == "" {
			v.DocsDir = "docs"
		}

		sh.SetDir(repoDir)
		err = sh.Command("git", "checkout", v.Branch).Run()
		if err != nil {
			return err
		}

		vDir := filepath.Join("docs", v.Branch)

		sh.SetDir(prjDir)
		err = sh.Command("cp", "-r", filepath.Join("repo", v.DocsDir), vDir).Run()
		if err != nil {
			return err
		}

		err := filepath.Walk(vDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil // skip
			}

			fmt.Println(path)
			// os.Exit(1)

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(data)

			page, err := parser.ReadFrom(buf)
			if err != nil {
				return err
			}

			metadata, err := page.Metadata()
			if err != nil {
				return err
			}

			aliases, ok, err := unstructured.NestedStringSlice(metadata, "aliases")
			if err != nil {
				return err
			}
			if ok {
				for i := range aliases {
					if strings.HasPrefix(aliases[i], "/") {
						aliases[i] = "/" + aliases[i]
					}
				}
				err = unstructured.SetNestedStringSlice(metadata, aliases, "aliases")
				if err != nil {
					return err
				}
			}

			yamlMetdata, err := yaml.Marshal(metadata)
			if err != nil {
				return err
			}

			content := page.Content()

			out := `---\n` + string(yamlMetdata) + `\n---\n` + string(content)
			return ioutil.WriteFile(path, []byte(out), 0755)
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}

		break
	}
	return nil
}
