package main

import (
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"log"
	"fmt"
	"io/ioutil"
)
const (
	ContentType = "binary/octet-stream"
)
func check(e error) {
	if e != nil {
		fmt.Println("Error", e.Error())
		panic(e)
	}
}

func main() {
	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}
	//	fmt.Println(aws.Regions)
	//	fmt.Println(aws.Regions)
	//	for n := aws.Regions {
	//		if n == "USEast" {
	//			r = n
	//		}
	//	}
	client := s3.New(auth, aws.USEast)
	//	bucket := client.Bucket(fmt.Sprintf("%s-%s", "lfs-server-go-objects", os.Getenv("GO_ENV")))
	bucket := client.Bucket("lfs-server-go-objects-test")
	//	fmt.Println("Bucket", bucket.Name)
//	berr := bucket.PutBucket(s3.Private)
//	check(berr)
	resp, err := bucket.ListBuckets()
	dat, err := ioutil.ReadFile("/tmp/abi.jpg")
	check(err)
	fmt.Println("Pushing abi.jpg")
	pErr := bucket.Put("abi.jpg", dat, ContentType, s3.PublicRead)
	check(pErr)
	if err != nil {
		log.Fatal(err)
	}
	items, err := bucket.List("", "", "", 1000)
	check(err)
	if len(items.Contents) > 0 {
		fmt.Println("Found items")
		for _, item := range items.Contents {
			fmt.Println("Item name", item.Key)
		}
	}
	fmt.Println("Deleting abi.jpg")
	k, ker := bucket.GetKey("abi.jpg")
	check(ker)
	derr := bucket.Del(fmt.Sprintf("%s",k.Key))
	check(derr)
	log.Print(fmt.Sprintf("%T %+v", resp.Buckets[0], resp.Buckets[0]))
}