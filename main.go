package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gohugoio/hugo/helpers"
	"github.com/spf13/cast"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	shell "github.com/codeskyblue/go-sh"
	"github.com/gohugoio/hugo/parser"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

		err := filepath.Walk(filepath.Join(prjDir, vDir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil // skip
			}

			fmt.Println(path)

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(data)

			page, err := parser.ReadFrom(buf)
			if err != nil {
				return err
			}

			content := page.Content()

			if strings.Index(string(content), "/docs") > -1 {
				re1 := regexp.MustCompile(`(\(/docs/)`)
				content = re1.ReplaceAll(content, []byte(`(/products/`+p.Name+`/`+v.Branch+`/`))

				re2 := regexp.MustCompile(`(\(/products/.*)(.md)(#.*)?\)`)
				content = re2.ReplaceAll(content, []byte(`${1}${3})`))

				content = bytes.ReplaceAll(content, []byte(`"/docs/images`), []byte(`"/products/`+p.Name+`/`+v.Branch+`/images`))
			}

			out := "---\n"
			frontmatter := page.FrontMatter()

			if len(frontmatter) != 0 && rune(frontmatter[0]) == '-' {
				var m2 yaml.MapSlice
				err = yaml.Unmarshal(frontmatter, &m2)
				if err != nil {
					return err
				}
				for i := range m2 {
					if sk, ok := m2[i].Key.(string); ok && sk == "aliases" {

						v2, ok := m2[i].Value.([]interface{})
						if !ok {
							continue
						}
						strSlice := make([]string, 0, len(v2))
						for _, v := range v2 {
							if str, ok := v.(string); ok {
								// make aliases abs path
								if !strings.HasPrefix(str, "/") {
									str = "/" + str
								}

								strSlice = append(strSlice, str)
							} else {
								continue
							}
						}
						m2[i].Value = strSlice
					} else if vv, changed := stringifyMapKeys(m2[i].Value); changed {
						m2[i].Value = vv
					}
				}

				d2, err := yaml.Marshal(m2)
				if err != nil {
					return err
				}
				out += string(d2)

			} else {
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
						if !strings.HasPrefix(aliases[i], "/") {
							aliases[i] = "/" + aliases[i]
						}
					}
					err = unstructured.SetNestedStringSlice(metadata, aliases, "aliases")
					if err != nil {
						return err
					}
				}

				metaYAML, err := yaml.Marshal(metadata)
				if err != nil {
					return err
				}
				out += string(metaYAML)
			}

			out = out + "---\n\n" + string(content)
			return ioutil.WriteFile(path, []byte(out), 0644)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// stringifyMapKeys recurses into in and changes all instances of
// map[interface{}]interface{} to map[string]interface{}. This is useful to
// work around the impedence mismatch between JSON and YAML unmarshaling that's
// described here: https://github.com/go-yaml/yaml/issues/139
//
// Inspired by https://github.com/stripe/stripe-mock, MIT licensed
func stringifyMapKeys(in interface{}) (interface{}, bool) {
	switch in := in.(type) {
	case []interface{}:
		for i, v := range in {
			if vv, replaced := stringifyMapKeys(v); replaced {
				in[i] = vv
			}
		}
	case map[interface{}]interface{}:
		res := make(map[string]interface{})
		var (
			ok  bool
			err error
		)
		for k, v := range in {
			var ks string

			if ks, ok = k.(string); !ok {
				ks, err = cast.ToStringE(k)
				if err != nil {
					ks = fmt.Sprintf("%v", k)
				}
				// TODO(bep) added in Hugo 0.37, remove some time in the future.
				helpers.DistinctFeedbackLog.Printf("WARNING: YAML data/frontmatter with keys of type %T is since Hugo 0.37 converted to strings", k)
			}
			if vv, replaced := stringifyMapKeys(v); replaced {
				res[ks] = vv
			} else {
				res[ks] = v
			}
		}
		return res, true
	}

	return nil, false
}
