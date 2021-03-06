package tgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

// ConnectionsResponse holds the response from `GET /network/connections`
type ConnectionsResponse struct {
	Incoming bool   `json:"incoming"`
	PeerID   string `json:"peer_id"`
	IDPoint  struct {
		Address string `json:"addr"`
		Port    int64  `json:"port"`
	} `json:"id_point"`
	RemoteSocketPort int64 `json:"remote_socket_port"`
	Versions         []struct {
		Name  string `json:"name"`
		Major int64  `json:"magor"`
		Minor int64  `json:"miner"`
	} `json:"versions"`
	Private       bool `json:"private"`
	LocalMetadata struct {
		DisableMempool bool `json:"disable_mempool"`
		PrivateNode    bool `json:"private_node"`
	} `json:"local_metadata"`
	RemoteMetadata struct {
		DisableMempool bool `json:"disable_mempool"`
		PrivateNode    bool `json:"private_node"`
	} `json:"remote_metadata"`
}

// GetConnections calls GET /network/connections
func (rpc *RPC) GetConnections() ([]ConnectionsResponse, error) {
	resp, err := rpc.Client.Get(fmt.Sprintf("%s/network/connections", rpc.URL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	cp := []ConnectionsResponse{}
	err = json.Unmarshal(respBytes, &cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// GetPeerID calls GET /network/connections/<peer_id>
func (rpc *RPC) GetPeerID(peerID string) (ConnectionsResponse, error) {
	resp, err := rpc.Client.Get(fmt.Sprintf("%s/network/connections/%s", rpc.URL, peerID))
	if err != nil {
		return ConnectionsResponse{}, err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ConnectionsResponse{}, err
	}
	cp := ConnectionsResponse{}
	err = json.Unmarshal(respBytes, &cp)
	if err != nil {
		return ConnectionsResponse{}, err
	}
	return cp, nil
}

// RemovePeers can be used to remove multiple peers at once
// Calls DELETE /network/connections/<peer_id>
func (rpc *RPC) RemovePeers(peers map[string]bool) ([]string, error) {
	processedPeers := []string{}
	for k, v := range peers {
		err := rpc.RemovePeer(k, v)
		if err != nil {
			return processedPeers, err
		}
		processedPeers = append(processedPeers, k)
	}
	return processedPeers, nil
}

// RemovePeer calls DELETE /network/connections/<peer_id>
func (rpc *RPC) RemovePeer(peerID string, wait bool) error {
	url := fmt.Sprintf("%s/network/connections/%s", rpc.URL, peerID)
	if wait {
		url = fmt.Sprintf("%s?wait", url)
	}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := rpc.Client.Do(req)
	defer resp.Body.Close()
	if resp.Status != "200 OK" {
		return fmt.Errorf("expected status '200 OK' got %s", resp.Status)
	}
	return nil
}

// ClearGreylist calls GET /network/greylist/clear
func (rpc *RPC) ClearGreylist() error {
	resp, err := rpc.Client.Get(fmt.Sprintf("%s/network/greylist/clear", rpc.URL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.Status != "200 OK" {
		return fmt.Errorf("expected status '200 OK' got %s", resp.Status)
	}
	return nil
}

// GetNetworkLog calls GET /network/log
// NOTE: Currently semi-bugged, closed after the first response
func (rpc *RPC) GetNetworkLog(waitTime time.Duration) error {
	url := fmt.Sprintf("%s/network/log", rpc.URL)
	resp, err := rpc.Client.Get(url)
	if err != nil {
		return err
	}
	go func() {
		time.Sleep(waitTime)
		resp.Body.Close()
	}()
	//defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return errors.New("expected object")
	}
	for decoder.More() {
		_, err := decoder.Token()
		if err != nil {
			return err
		}
		var v interface{}
		err = decoder.Decode(&v)
		if err != nil {
			return err
		}
		fmt.Printf("%+v\n", v)
	}
	return nil
}

type NetworkPeers struct {
	PublicKeyHash string
	Score         int64 `json:"score"`
	Trusted       bool  `json:"trusted"`
	ConnMetadata  struct {
		DisableMempool bool `json:"disable_mempool"`
		PrivateNode    bool `json:"private_node"`
	} `json:"conn_metadata"`
	State       string `json:"state"`
	ReachableAt struct {
		Addr string `json:"addr"`
		Port int64  `json:"port"`
	} `json:"reachable_at"`
	Stat struct {
		TotalSent      int64 `json:"total_sent"`
		TotalRecv      int64 `json:"total_recv"`
		CurrentInflow  int64 `json:"current_inflow"`
		CurrentOutflow int64 `json:"current_outflow"`
	} `json:"stat"`
	LastFailedConnection struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port"`
		Timestamp int64
	} `json:"last_failed_connection,omitempty"`
	LastRejectedConnection struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port"`
		Timestamp int64
	} `json:"last_rejected_connection,omitempty"`
	LastEstablishedConnection struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port"`
		Timestamp int64
	} `json:"last_established_connection,omitempty"`
	LastDisconnection struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port"`
		Timestamp int64
	} `json:"last_disconnection,omitempty"`
	LastSeen struct {
		Addr      string `json:"addr"`
		Port      string `json:"port"`
		Timestamp int64
	} `json:"last_seen,omitempty"`
	LastMiss struct {
		Addr      string `json:"addr"`
		Port      string `json:"port"`
		Timestamp int64
	} `json:"last_miss,omitempty"`
}

// GetNetworkPeers calls GET /network/peers
//TODO: implement filter
func (rpc *RPC) GetNetworkPeers() error {
	url := fmt.Sprintf("%s/network/peers", rpc.URL)
	resp, err := rpc.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var raw interface{}
	err = json.Unmarshal(respBytes, &raw)
	if err != nil {
		return err
	}
	peers := [][]NetworkPeers{}
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &peers)
	if err != nil {
		return err
	}
	return nil
}

type NetworkPeer struct {
	Score        int64 `json:"score,string"`
	Trusted      bool  `json:"trusted"`
	ConnMetadata struct {
		DisableMempool bool `json:"disable_mempool"`
		PrivateNode    bool `json:"private_node"`
	} `json:"conn_metadata"`
	State       string `json:"state"`
	ReachableAt struct {
		Addr string `json:"addr"`
		Port int64  `json:"port"`
	} `json:"reachable_at"`
	Stat struct {
		TotalSent      string `json:"total_sent"`
		TotalRecv      string `json:"total_recv"`
		CurrentInflow  int64  `json:"current_inflow,string"`
		CurrentOutflow int64  `json:"current_outflow,string"`
	} `json:"stat"`
	LastFailedConnection struct {
		Addr      string `json:"addr"`
		Port      string `json:"port,omitempty"`
		Timestamp int64
	} `json:"last_failed_connection,omitempty"`
	LastRejectedConnection []struct {
		Addr string `json:"addr"`
		Port string `json:"port,omitempty"`
		//Timestamp int64
	} `json:"last_rejected_connection,omitempty"`
	LastEstablishedConnection struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port,omitempty"`
		Timestamp int64
	} `json:"last_established_connection,omitempty"`
	LastDisconnection struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port,omitempty"`
		Timestamp int64
	} `json:"last_disconnection,omitempty"`
	LastSeen struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port,omitempty"`
		Timestamp int64
	} `json:"last_seen,omitempty"`
	LastMiss struct {
		Addr      string `json:"addr"`
		Port      int64  `json:"port,omitempty"`
		Timestamp int64
	} `json:"last_miss,omitempty"`
}

func (rpc *RPC) GetNetworkPeer(peerID string) error {
	url := fmt.Sprintf("%s/network/peers/%s", rpc.URL, peerID)
	resp, err := rpc.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`(":\s*)([\d\.]+)(\s*[,}])`)
	respBytes = re.ReplaceAll(respBytes, []byte(`$1"$2"$3`))
	peer := NetworkPeer{}
	err = json.Unmarshal(respBytes, &peer)
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", peer)
	return nil
}
