package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"fmt"
	"bytes"
	"encoding/json"

	"github.com/pquerna/ffjson/ffjson"
	log "github.com/ikenchina/golog"

	"github.com/Spefer/common/libs/proto"
	"strings"
)

type HttpServer struct {
	Config ConfigHttp
}

func InitHTTP() (err error) {
	// http listen
	h := new(HttpServer)
	h.Config = Conf.Http

	return h.Init()
}

func UnInitHTTP() {

}

func (h *HttpServer) Init() (err error) {
	for _, b := range h.Config.Bind {
		httpServeMux := http.NewServeMux()
		httpServeMux.HandleFunc("/push/user", h.Push)
		httpServeMux.HandleFunc("/push/users", h.Pushs)
		httpServeMux.HandleFunc("/push/topic", h.PushTopic)
		httpServeMux.HandleFunc("/push/all", h.PushAll)

		httpServeMux.HandleFunc("/server/del", h.Server)
		httpServeMux.HandleFunc("/server/count", h.ServerCount)

		httpServeMux.HandleFunc("/users", h.UsersInfo)
		httpServeMux.HandleFunc("/info", h.Info)

		log.Infof("start http listen : %v", b)

		// panic directly
		go h.httpListen(httpServeMux, b.Network, b.Addr)
	}

	return
}

func (h *HttpServer) httpListen(mux *http.ServeMux, network, addr string) {

	httpServer := &http.Server{Handler: mux}
	httpServer.SetKeepAlivesEnabled(true)
	l, err := net.Listen(network, addr)
	if err != nil {
		log.Fatalf("HttpServer listen (%s@%s) failed : %v", network, addr, err)
		panic(err)
	}
	if err := httpServer.Serve(l); err != nil {
		log.Fatalf("HttpServer.Serve() failed : %v", err)
		panic(err)
	}
}

// ************** push handler *******************//
/*
{
	"userid" : "xxx",
	"ensure" : true,
	"expire" : xxx,
	"redomax" : -1,
	"body" : "yyy"	
}
*/
// push message to a user
func (h *HttpServer) Push(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var (
		serverId  int32
		bodyBytes []byte
		err       error
	)
	defer r.Body.Close()
	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg := &proto.PushMsg{}
	err = ffjson.Unmarshal(bodyBytes, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serverId, err = DefaultServer.GetBucket(msg.UserId).GetUserServer(msg.UserId)
	if err != nil {
		if msg.Ensure {
			err = SaveMsg(msg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

		return
	}

	if err = PushMsg(serverId, msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	return
}

/*
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
*/
type MPushREQ struct {
	UserId []string
	Msg    *proto.PushMsg
}

// push message to many users
func (h *HttpServer) Pushs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var (
		bodyBytes []byte
		serverId  int32
		err       error
	)
	defer r.Body.Close()
	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqmsg := &MPushREQ{}
	//log.Debugf("before unmarshal: %s",bodyBytes)
	err = ffjson.Unmarshal(bodyBytes, reqmsg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debugf("msg: %s;body:%s", strings.Join(reqmsg.UserId, ","), reqmsg.Msg.Body)
	type ResponseJson struct {
		Success []string //userid
		Failed  []string //userid
		Msg     []string
	}
	RespJson := new(ResponseJson)

	// map : serverid --> users
	servermap := make(map[int32][]string)
	for _, u := range reqmsg.UserId {
		serverId, err = DefaultServer.GetBucket(u).GetUserServer(u)
		log.Debugf("server id:%d, userid = %s, err=%s", serverId, u, err)
		if err != nil {
			if reqmsg.Msg.Ensure {
				reqmsg.Msg.UserId = u
				err = SaveMsg(reqmsg.Msg)
				if err != nil {
					RespJson.Failed = append(RespJson.Failed, u)
					RespJson.Msg = append(RespJson.Msg, err.Error())
				} else {
					RespJson.Success = append(RespJson.Success, u)
				}
			}
		} else {
			servermap[serverId] = append(servermap[serverId], u)
		}
	}

	for sid, uids := range servermap {
		log.Debugf("servermap -> server id:%d, userids = %s", sid, strings.Join(uids, ","))
		for _, uid := range uids {
			reqmsg.Msg.UserId = uid
			log.Debugf("current push to uid = %s", uid)
			cloneMsg := &proto.PushMsg{
				UserId: uid,
				Ensure: reqmsg.Msg.Ensure,
				Expire: reqmsg.Msg.Expire,
				RedoMax: reqmsg.Msg.RedoMax,
				Body: reqmsg.Msg.Body}
			// @TODO optimize : change to MPushMsg
			if err = PushMsg(sid, cloneMsg); err != nil {
				RespJson.Failed = append(RespJson.Failed, uid)
				RespJson.Msg = append(RespJson.Msg, err.Error())

			} else {
				RespJson.Success = append(RespJson.Success, uid)
			}
		}
	}
	datas, err := ffjson.Marshal(RespJson)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	lend, err := w.Write(datas)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if lend != len(datas) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return
}

/*
{
	"topic" : xxx,
	"msg":{
		"body" : "yyy"
	}
}
*/
type PushTopicREQ struct {
	Topic string
	Msg   *proto.PushMsg
}

// push message of topic to users who subscribe the topic
func (h *HttpServer) PushTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var (
		bodyBytes []byte
		err       error
	)
	defer r.Body.Close()
	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqmsg := &PushTopicREQ{}
	fmt.Printf("%s", bodyBytes)
	log.Debugf("%s", bodyBytes)
	err = ffjson.Unmarshal(bodyBytes, reqmsg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Errorf("Error on Unmarshal following content to PushTopicREQ\n%s", bodyBytes)
		return
	}
	fmt.Printf("\nbody: %s", reqmsg.Msg.Body)
	log.Debugf("\nbody: %s", reqmsg.Msg.Body)
	servers := DefaultServer.GetTopicServers(reqmsg.Topic)
	if len(servers) == 0 {
		http.Error(w, "no servers sub the topic", http.StatusInternalServerError)
		return
	}

	if err = DefaultMQ.BroadcastTopic(servers, reqmsg.Topic, reqmsg.Msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	return
}

/*
{
	"body" : "yyy"
}
*/
// broadcast :  push message to all users 
func (h *HttpServer) PushAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var (
		bodyBytes []byte
		err       error
	)
	defer r.Body.Close()
	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := &proto.PushMsg{}
	err = ffjson.Unmarshal(bodyBytes, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := DefaultMQ.Broadcast(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	return
}

func (h *HttpServer) Server(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		h.DeleteServer(w, r)
	}
}

// delete server : when comet was crashed, need to call this api to delete server then restart the comet
func (h *HttpServer) DeleteServer(w http.ResponseWriter, r *http.Request) {

	var (
		err       error
		serverStr = r.URL.Query().Get("id")
		server    int64
	)
	if server, err = strconv.ParseInt(serverStr, 10, 32); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = DefaultServer.DeleteServer(int32(server)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// report information of users :
// request : http://localhost:8880/users
// response : {"servers":[{"id":1,"users":[{"Id":32,"Topics":[1,2,3]}]}]}
func (h *HttpServer) UsersInfo(w http.ResponseWriter, r *http.Request) {

	datas := new(bytes.Buffer)
	servers := DefaultServer.GetServers()

	datas.WriteString(fmt.Sprintf(`{"servers":[`))
	firstServer := true

	for _, sid := range servers {
		if firstServer {
			datas.WriteString(fmt.Sprintf(`{"id":%d,"users":`, sid))
			firstServer = false
		} else {
			datas.WriteString(fmt.Sprintf(`,{"id":%d,"users":`, sid))
		}
		users := DefaultServer.GetUsers(sid)

		udata, err := json.Marshal(users)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		datas.Write(udata)
		datas.WriteString("}")
	}

	datas.WriteString("]}")
	w.Write(datas.Bytes())
}

// get amount of user of server
// request : http://localhost:8880/server/count?id=1
// response: {"serverid":1, "count":10}
func (h *HttpServer) ServerCount(w http.ResponseWriter, r *http.Request) {

	var (
		serverid  int
		err       error
		serverStr = r.URL.Query().Get("id")
		count     int64
	)

	serverStr = strings.TrimSpace(serverStr)
	serverid, err = strconv.Atoi(serverStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	count = DefaultServer.GetServerSize(int32(serverid))

	resp := fmt.Sprintf(`{"serverid":%d, "count":%d}`, serverid, count)
	w.Write([]byte(resp))
}

type InfoResp struct {
	CleanDataSize int
}

func (h *HttpServer) Info(w http.ResponseWriter, r *http.Request) {

	infoRES := new(InfoResp)

	for _, bk := range DefaultServer.Buckets {
		infoRES.CleanDataSize += bk.GetCleanDataSize()
	}
	resp, err := json.Marshal(infoRES)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(resp)

}
