package main

func perror(err error) {
	if err != nil {
		logger.Log(kv{"fn": "perror", "msg": err.Error()})
		panic(err)
	}
}
