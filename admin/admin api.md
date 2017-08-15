## API
Admin Server Restful API
### API list:
- /push/user
- /push/users
- /push/topic
- /push/all
- /server/del
- /server/count
- /users
- /info


### Specific Details
#### /push/user 
Request Body:

    {
    	"userid" : xxx,
    	"ensure" : true,
    	"expire" : xxx,
    	"redomax" : -1,
    	"body" : "yyy"	
    }
    
#### /push/users
Request Body:

    {
        "userid" : [
            xxx, yyy, zzz
        ],
        "msg":{
            "ensure" : true,
            "expire" : xxx,
            "redomax" : -1,
            "body" : "yyy"	
        }
    }
    
    
#### /push/topic
Request Body:

    {
        "topic" : [
            xxx, yyy, zzz
        ],
        "msg":{
            "body" : "yyy"	
        }
    }
    
#### /push/all
Request Body:

    {
        "body" : "yyy"	
    }
    
#### /server/del
Request Body:
    
    // like: request : http://localhost:8880/server/del?id=1
    // delete server : when comet was crashed, need to call this api to delete server then restart the comet
    
#### /server/count
Request Body:

    // get amount of user of server
    // request : http://localhost:8880/server/count?id=1
    // response: {"serverid":1, "count":10}
    
#### /users
    // report information of users : 
    // request : http://localhost:8880/users
    // response : {"servers":[{"id":1,"users":[{"Id":32,"Topics":[1,2,3]}]}]}

#### /info
    Not available now
    
    