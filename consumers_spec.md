# user\_service Spec

> Spec for consuming user authorization of a given project

Given a URL, it MUST respond with a boolean value as `access` to a request with the following parameters:

<pre>
username: String
project: String
action: String
</pre>

Allowed actions are: 

* download
* push
* force\_push
* admin

download
---
download allows a user to only download, meaning that pushing is disallowed for this user

push
---
push allows for downloading and pushing to a given project

force\_push
---
force\_push Allows for downloading, pushing, and push -f (force) pushing to the given project
 
admin
---
admin allows for any and all of the above actions, and anything else.  Should always return `true` for any action. 


Example request: 

```json
{ 
  "username": "jsmith",
  "project": "myproject",
  "action": "download"
}
```

Response:

```json
{ 
  "access": true, 
  "status": "Optional Status like 'success'", 
  "message": "Optional message like 'has access'" 
}
```

 