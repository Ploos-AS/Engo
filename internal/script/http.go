package script

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClient struct {
	allowed map[string]bool
	client *http.Client
	maxBody int64
}

func NewHTTPClient(hosts []string,timeout time.Duration,maxBody int64)*HTTPClient{
	allowed:=make(map[string]bool,len(hosts))
	for _,h:=range hosts{h=strings.ToLower(strings.TrimSpace(h));if h!=""{allowed[h]=true}}
	h:=&HTTPClient{allowed:allowed,maxBody:maxBody}
	h.client=&http.Client{Timeout:timeout,CheckRedirect:func(req *http.Request,via []*http.Request)error{
		if len(via)>=5{return fmt.Errorf("too many redirects")}
		return h.validateURL(req.URL)
	}}
	return h
}
func (h *HTTPClient) Enabled()bool{return len(h.allowed)>0}
func (h *HTTPClient) Get(raw string)(map[string]interface{},error){
	u,err:=url.Parse(raw);if err!=nil{return nil,err};if err:=h.validateURL(u);err!=nil{return nil,err}
	resp,err:=h.client.Get(u.String());if err!=nil{return nil,err};defer resp.Body.Close()
	body,err:=io.ReadAll(io.LimitReader(resp.Body,h.maxBody+1));if err!=nil{return nil,err}
	if int64(len(body))>h.maxBody{return nil,fmt.Errorf("HTTP response exceeds %d bytes",h.maxBody)}
	return map[string]interface{}{"status":int64(resp.StatusCode),"body":string(body),"content_type":resp.Header.Get("Content-Type")},nil
}
func (h *HTTPClient) validateURL(u *url.URL)error{
	if u.Scheme!="https"{return fmt.Errorf("HTTP capability requires https")}
	host:=strings.ToLower(u.Hostname());if !h.allowed[host]{return fmt.Errorf("HTTP host %q is not allowed",host)}
	ips,err:=net.LookupIP(host);if err!=nil{return fmt.Errorf("resolve HTTP host: %w",err)}
	for _,ip:=range ips{if blockedIP(ip){return fmt.Errorf("HTTP host resolves to blocked address")}
	return nil
}
func blockedIP(ip net.IP)bool{
	return ip.IsLoopback()||ip.IsPrivate()||ip.IsUnspecified()||ip.IsMulticast()||ip.IsLinkLocalUnicast()||ip.IsLinkLocalMulticast()
}
