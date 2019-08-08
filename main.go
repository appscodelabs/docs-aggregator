package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
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

func processProduct(name string, p Product) error {

	return nil
}
