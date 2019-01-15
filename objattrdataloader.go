package dataloaders

import (
	"fmt"
	"sync"
)

func NewObjAttrDataLoader(initLoaders ObjAttrDataLoaderInits) *ObjAttrDataLoader {
	if initLoaders == nil {
		initLoaders = ObjAttrDataLoaderInits{}
	}
	return &ObjAttrDataLoader{
		initLoaders: initLoaders,
		loaders:     ObjAttrDataLoaders{},
	}
}

type ObjAttrDataLoader struct {
	// Init loader when uninitialized attribute is called.
	initLoaders ObjAttrDataLoaderInits

	// The loaders & caches.
	loaders ObjAttrDataLoaders

	// Mutex to prevent races.
	mu sync.Mutex
}

type ObjectType interface{}

// ObjAttrDataLoader initializers map
type ObjAttrDataLoaderInits map[ObjectType]func() *AttrDataLoader

// AttributeDataLoaders map
type ObjAttrDataLoaders map[ObjectType]*AttrDataLoader

func (l *ObjAttrDataLoader) Load(objectType ObjectType, attribute Attribute, key Key) (Value, error) {
	if loader := l.loader(objectType); loader != nil {
		return loader.Load(attribute, key)
	} else {
		return nil, NewObjTypeNotRegError(fmt.Sprintf("no dataloader for objectType '%s' registered", objectType))
	}
}

func (l *ObjAttrDataLoader) LoadAll(objectType ObjectType, attribute Attribute, keys []Key) ([]Value, []error) {
	if loader := l.loader(objectType); loader != nil {
		return loader.LoadAll(attribute, keys)
	} else {
		return nil, []error{NewObjTypeNotRegError(fmt.Sprintf("no dataloader for objectType '%s' registered", objectType))}
	}
}

// Prime the cache with the provided objectType, attribute, key and value.
// If the key already exists, no change is made
// and false is returned. Returns false if attribute not registered.
// (To forcefully prime the cache, use l.ForcePrime().)
func (l *ObjAttrDataLoader) Prime(objectType ObjectType, attribute Attribute, key Key, value Value) bool {
	return l.prime(objectType, attribute, key, value, false)
}

// Forcefully prime the cache with the provided objectType, attribute, key and value.
func (l *ObjAttrDataLoader) ForcePrime(objectType ObjectType, attribute Attribute, key Key, value Value) bool {
	return l.prime(objectType, attribute, key, value, true)
}

func (l *ObjAttrDataLoader) prime(objectType ObjectType, attribute Attribute, key Key, value Value, forcePrime bool) bool {
	if loader := l.loader(objectType); loader != nil {
		return loader.prime(attribute, key, value, forcePrime)
	}
	return false
}

// Clear the value at key at attribute for objectType from the cache, if it exists.
func (l *ObjAttrDataLoader) Clear(objectType ObjectType, attribute Attribute, key Key) *ObjAttrDataLoader {
	if loader := l.loader(objectType); loader != nil {
		loader.Clear(attribute, key)
	}
	return l
}

// Returns the dataloader of the objectType.
// Initializes the dataloader if not exists and initializer is registered.
func (l *ObjAttrDataLoader) loader(objectType ObjectType) *AttrDataLoader {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Check loader of attribute is initialized.
	if loader, exists := l.loaders[objectType]; exists {
		return loader
	} else { // Init if init func registered.
		if loaderInit, exists := l.initLoaders[objectType]; exists {
			// create loader
			loader = loaderInit()
			// remove init func, since no longer needed
			l.initLoaders[objectType] = nil
			// set loader
			l.loaders[objectType] = loader
			// return the loader
			return loader
		}
	}
	// Loader not registered.
	return nil
}

// Occurs when an unregistered object type is requested.
type ObjTypeNotRegError struct {
	msg string
}

func (e *ObjTypeNotRegError) Error() string {
	return e.msg
}

func NewObjTypeNotRegError(msg string) error {
	return &ObjTypeNotRegError{msg: msg}
}

// Returns true if the error occourd when running the loader to resolve data.
func IsLoadingError(err error) bool {
	if err != nil {
		switch err.(type) {
		case *ObjTypeNotRegError:
			return false
		case *AttrNotRegError:
			return false
		}
	}
	return true
}
