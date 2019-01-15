package dataloaders

import (
	"fmt"
	"sync"
)

func NewAttrDataLoader(initLoaders AttrDataLoaderInits, propagators ValuePropagators) *AttrDataLoader {
	if initLoaders == nil {
		initLoaders = AttrDataLoaderInits{}
	}
	if propagators == nil {
		propagators = ValuePropagators{}
	}
	return &AttrDataLoader{
		initLoaders: initLoaders,
		propagators: propagators,
		loaders:     AttrDataLoaders{},
	}
}

type AttrDataLoader struct {
	// Init loader when uninitialized attribute is called.
	initLoaders AttrDataLoaderInits

	// The loaders & caches.
	loaders AttrDataLoaders

	// See ValuePropagator type description.
	propagators ValuePropagators

	// Mutex to prevent races.
	mu sync.Mutex
}

// AttrDataLoaderInits initializers map
type AttrDataLoaderInits map[Attribute]func() *DataLoader

// AttrDataLoaders map
type AttrDataLoaders map[Attribute]*DataLoader

// ValuePropagators map
type ValuePropagators map[Attribute]ValuePropagator

// 	Parameters:
// 		loadedValue - the just loaded value
// 		l - the attribute loader (use the functions in it)
//
// ValuePropagators are defined to propagate the cache with already loaded objects
// which contain an attribute also registered in this AttrDataLoader.
// The ValuePropagator for the attribute is executed directly after the Value was loaded.
//
// 	Why and how is a ValuePropagator used?:
// 		Use ValuePropagators to propagate keys in the cache with loaded objects containing these attributes.
// 		Here is an example:
// 			An UserAccount is loaded by the attribute id.
// 			The loaded UserAccount also contains the email address field which might also be used to load UserAccounts.
// 			So instead of maybe completely loading the UserAccount by email again,
// 			we pre-allocate the keys (e.g. email) with already loaded Values (e.g. UserAccount) containing the attribute (e.g. email).
// 		How?:
// 			You can propagate/prime a cache using l.Prime(attribute, key, value).
type ValuePropagator func(loadedValue Value, l *AttrDataLoader)
type Attribute interface{}

func (l *AttrDataLoader) Load(attribute Attribute, key Key) (Value, error) {
	if loader := l.loader(attribute); loader != nil {
		value, err := loader.Load(key)
		if err == nil {
			l.RunPropagator(value, attribute)
		}
		return value, err
	} else {
		return nil, NewAttrNotRegError(fmt.Sprintf("no dataloader for attribute '%s' registered", attribute))
	}
}

func (l *AttrDataLoader) LoadAll(attribute Attribute, keys []Key) ([]Value, []error) {
	if loader := l.loader(attribute); loader != nil {
		values, errs := loader.LoadAll(keys)
		for val := range values {
			l.RunPropagator(val, attribute)
		}
		return values, errs
	} else {
		return nil, []error{NewAttrNotRegError(fmt.Sprintf("no dataloader for attribute '%s' registered", attribute))}
	}
}

// Runs the propagator if registered for the attribute.
func (l *AttrDataLoader) RunPropagator(value Value, attribute Attribute) {
	propagator, exists := l.propagators[attribute]
	if exists {
		propagator(value, l)
	}
}

// Prime the cache with the provided attribute, key and value.
// If the key already exists, no change is made
// and false is returned. Returns false if attribute not registered.
// (To forcefully prime the cache, use l.ForcePrime().)
func (l *AttrDataLoader) Prime(attribute Attribute, key Key, value Value) bool {
	return l.prime(attribute, key, value, false)
}

// Forcefully prime the cache with the provided attribute, key and value.
func (l *AttrDataLoader) ForcePrime(attribute Attribute, key Key, value Value) {
	l.prime(attribute, key, value, true)
}

func (l *AttrDataLoader) prime(attribute Attribute, key Key, value Value, forcePrime bool) bool {
	if loader := l.loader(attribute); loader != nil {
		return loader.prime(key, value, forcePrime)
	}
	return false
}

// Clear the value at key at attribute from the cache, if it exists.
func (l *AttrDataLoader) Clear(attribute Attribute, key Key) *AttrDataLoader {
	if loader := l.loader(attribute); loader != nil {
		loader.Clear(key)
	}
	return l
}

// Returns the dataloader of the attribute.
// Initializes the dataloader if not exists and initializer is registered.
func (l *AttrDataLoader) loader(attribute Attribute) *DataLoader {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Check loader of attribute is initialized.
	if loader, exists := l.loaders[attribute]; exists {
		return loader
	} else { // Init if init func registered.
		if loaderInit, exists := l.initLoaders[attribute]; exists {
			// create loader
			loader = loaderInit()
			// remove init func, since no longer needed
			l.initLoaders[attribute] = nil
			// set loader
			l.loaders[attribute] = loader
			// return the loader
			return loader
		}
	}
	// Loader not registered.
	return nil
}

// Occurs when an unregistered attribute is requested.
type AttrNotRegError struct {
	msg string
}

func (e *AttrNotRegError) Error() string {
	return e.msg
}

func NewAttrNotRegError(msg string) error {
	return &AttrNotRegError{msg: msg}
}
