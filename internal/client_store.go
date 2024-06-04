package internal

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type DuplicateClientIdError struct {
	Id uint32
}

func (e *DuplicateClientIdError) Error() string {
	return fmt.Sprintf("Attempted to create client with duplicate ID %d", e.Id)
}

type MissingClientIdError struct {
	Id uint32
}

func (e *MissingClientIdError) Error() string {
	return fmt.Sprintf("Missing client with id=%d", e.Id)
}

type ClientConnectionMetadata struct {
	Mut                    sync.RWMutex
	IsConnected            bool
	ClientHandlerName      string
	DestinationHandlerName string
	LastClientMsgTime      int64
	LastDestinationMsgTime int64
}

type ClientStore struct {
	nextClientId atomic.Uint32

	mut_clientConnections sync.RWMutex
	clientConnections     map[uint32]*ClientConnectionMetadata
}

func (store *ClientStore) GetNewClientId() uint32 {
	return store.nextClientId.Add(1)
}

func (store *ClientStore) HasClient(clientId uint32) bool {
	store.mut_clientConnections.RLock()
	defer store.mut_clientConnections.RUnlock()

	_, has := store.clientConnections[clientId]
	return has
}

func (store *ClientStore) CreateClient(clientId uint32, clientHandlerName string, timestamp int64) error {
	store.mut_clientConnections.Lock()
	defer store.mut_clientConnections.Unlock()

	if _, has := store.clientConnections[clientId]; has {
		return &DuplicateClientIdError{Id: clientId}
	}

	store.clientConnections[clientId] = &ClientConnectionMetadata{
		Mut:                    sync.RWMutex{},
		IsConnected:            false,
		ClientHandlerName:      clientHandlerName,
		DestinationHandlerName: "",
		LastClientMsgTime:      timestamp,
		LastDestinationMsgTime: timestamp,
	}

	return nil
}

func (store *ClientStore) Connect(clientId uint32) error {
	store.mut_clientConnections.RLock()
	defer store.mut_clientConnections.RUnlock()

	connection, has := store.clientConnections[clientId]
	if !has {
		return &MissingClientIdError{Id: clientId}
	}

	connection.Mut.Lock()
	defer connection.Mut.Unlock()

	connection.IsConnected = true
	return nil
}

func (store *ClientStore) SetDestinationHandlerName(clientId uint32, destinationHandlerName string) error {
	store.mut_clientConnections.RLock()
	defer store.mut_clientConnections.RUnlock()

	connection, has := store.clientConnections[clientId]
	if !has {
		return &MissingClientIdError{Id: clientId}
	}

	connection.Mut.Lock()
	defer connection.Mut.Unlock()

	connection.DestinationHandlerName = destinationHandlerName
	return nil
}

func (store *ClientStore) GetDestinationHandlerName(clientId uint32) (string, error) {
	store.mut_clientConnections.RLock()
	defer store.mut_clientConnections.RUnlock()

	connection, has := store.clientConnections[clientId]
	if !has {
		return "", &MissingClientIdError{Id: clientId}
	}

	connection.Mut.RLock()
	defer connection.Mut.RUnlock()

	return connection.DestinationHandlerName, nil
}

func (store *ClientStore) GetClientHandlerName(clientId uint32) (string, error) {
	store.mut_clientConnections.RLock()
	defer store.mut_clientConnections.RUnlock()

	connection, has := store.clientConnections[clientId]
	if !has {
		return "", &MissingClientIdError{Id: clientId}
	}

	connection.Mut.RLock()
	defer connection.Mut.RUnlock()

	return connection.ClientHandlerName, nil
}

func (store *ClientStore) SetClientRecvTimestamp(clientId uint32, timestamp int64) error {
	store.mut_clientConnections.RLock()
	defer store.mut_clientConnections.RUnlock()

	connection, has := store.clientConnections[clientId]
	if !has {
		return &MissingClientIdError{Id: clientId}
	}

	connection.Mut.Lock()
	defer connection.Mut.Unlock()

	connection.LastClientMsgTime = timestamp
	return nil
}

func (store *ClientStore) SetDestinationRecvTimestamp(clientId uint32, timestamp int64) error {
	store.mut_clientConnections.RLock()
	defer store.mut_clientConnections.RUnlock()

	connection, has := store.clientConnections[clientId]
	if !has {
		return &MissingClientIdError{Id: clientId}
	}

	connection.Mut.Lock()
	defer connection.Mut.Unlock()

	connection.LastDestinationMsgTime = timestamp
	return nil
}
