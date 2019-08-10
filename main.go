package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

func main()  {
	if err := dostuff(); err != nil {
		log.Fatalln(err)
	}
}

func dostuff() error {
	data, err := ioutil.ReadFile("/home/tamal/AppsCode/Source/appscode_web/products.json")
	if err != nil {
		return err
	}

	var cfg DocAggregator
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}

	for name, p := range cfg.Products {
		err = processProduct(name, p)
		if err != nil {
			return err
		}
		break // skip
	}

	return nil
}

/*
session := sh.NewSession()
session.SetEnv("BUILD_ID", "123")
session.SetDir("/")
# then call cmd
session.Command("echo", "hello").Run()
# set ShowCMD to true for easily debug
session.ShowCMD = true
*/
func processProduct(name string, p Product) error {
	dir, err := ioutil.TempDir("/home/tamal/Desktop/docs", "aggr")
	if err != nil {
		return err
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}


	return nil
}
