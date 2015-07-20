package main

import (
	"fmt"
	"gopkg.in/redis.v3"
	"strconv"
)

func main() {
	//usersK := "Users"
	//usernameK := "username"
	//passK := "password"
	// projectsK := "projects"
	objectsK := "all:objects"
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0, // use default DB
	})
	obj1 := "aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019g"
	obj2 := "aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019a"
	obj3 := "aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019h"

	//client.HSet(projectsK, "project", "project1").Result()
	// Create oid refs with values
	client.HMSet(obj1, "size", strconv.Itoa(100)).Result()
	client.HMSet(obj2, "size", "200").Result()
	client.HMSet(obj3, "size", "300").Result()
	// Add the objects to the project
	client.SAdd("objects:project1", obj1, obj2).Result()
	// Create the project -> OID refs
	// Add the object references to the global object pool
	client.SAdd(objectsK, obj1, obj2).Result()
	// Add another object
	client.SAdd(objectsK, obj3).Result()
//	client.HMSet("project1", "oids", objectsK).Result()
//	client.HMSet("project1", "oids", objectsK).Result()
	// get object stuff
	obj1_data, _ := client.HGet(obj1, "size").Result()
	fmt.Println("Object1 size", obj1_data)
	oops, oerr := client.SMembers("objects:notthere").Result()
	fmt.Println("OOPS", oops)
	fmt.Println("OOPS ERR", oerr)
	// get objects belonging to a project
	prj1_data, _ := client.SMembers("objects:project1").Result()
	fmt.Printf("%s objects %s\n", "project1", prj1_data)
	// get all objects
	all_objs, _ := client.SMembers("all:objects").Result()
	fmt.Printf("%s objects %s\n", "all", all_objs)
	// remove obj1
	client.SRem("all:objects", obj1).Result()
	objs_now, _ := client.SMembers("all:objects").Result()
	fmt.Printf("Removed one from %s objects, now there are 2 left %s\n", "all:objects", objs_now)
}