# DataLoaders
> Efficient DataLoader types making creating DataLoaders fun!

[![GoDoc][godoc-img]][godoc]
<!-- [![Sourcegraph][sourcegraph-img]][sourcegraph] -->


Using this package you can efficiently create
request based DataLoaders which will only be created when they are needed
and also define one root DataLoader containing all loaders needed per object and attribute.

## Installation
`go get -u github.com/robinbraemer/dataloaders`

## The DataLoaders
It is intended only to be used for short lived DataLoaders
(i.e. DataLoaders that ony exsist for the life of an http request).

There are multiple types of DataLoaders included which can be used upon each other.
Please see all DataLoader types before choosing one.

#### Default DataLoader
The default DataLoader is implemented by all other DataLoader types
and can load data by only one type of attribute.
What that exactly means you will see in the following DataLoader types.
It is recommended to use the other DataLoader types and not the default one
because they are more efficient and extendable.

A default DataLoader is beeing created like this example:
```go
dataloaders.NewDataLoader(100, 1*time.Millisecond,
    func(keys []Key) (values []Value, errors []error) {
        // load data using keys...
    }) 
```

#### Attribute DataLoader
*Please understand the **default DataLoader** first if you haven't already.*

Here it starts to get efficient and extendable.
The attribute DataLoader allows loading data by multiple attributes.
For example you can load an Account object by ID, email, name, etc.
DataLoader types using an attribute DataLoader underlying are efficient
because the DataLoader only gets initialized when it is needed.
Whe only want to create the DataLoaders which are needed for example
in an api request.

Then there are **value propagators** which **spread** the **loaded value** in the cache.
For example if it has loaded an Account by *id*, the registered propagator for attribute *id*
is being executed internally before returning the value to the DataLoader caller.
Inside you could prime the cache key/attribute *email* (primary) which the Account have so it don't
need to fetch the same Account unnecessarily again by *email* because it then can be found in the cache.

It is considered to create this type of DataLoader once per object type you have (e.g. Account).

A attribute DataLoader is beeing created like this example:
```go
dataloaders.NewAttrDataLoader(AttrDataLoaderInits{
    "id": func() *DataLoader {
        return NewDataLoader(100, 1*time.Millisecond,
            func(keys []Key) (values []Value, errors []error) {
                // load data using ids...
            })
    },
    "email": func() *DataLoader {
        return NewDataLoader(100, 1*time.Millisecond,
            func(keys []Key) (values []Value, errors []error) {
                // load data using emails...
            })
    },
}, ValuePropagators{
    "id": func(v Value, l *AttrDataLoader) {
        // prime the cache with key v.Email = v
    },
    "email": func(v Value, l *AttrDataLoader) {
        // prime the cache with key v.ID = v
    },
}, )
```

#### Object Attribute DataLoader
*Please understand the **attribute DataLoader** first if you haven't already.*

The object attribute DataLoader implements the attribute DataLoader.
Now you can create this DataLoader once in your application and
define all your object types in it and still be efficient.
Like mentioned before, still every DataLoader is only initialized when it is needed.

A object attribute DataLoader is beeing created like this example:
```go
dataloaders.NewObjAttrDataLoader(ObjAttrDataLoaderInits{
    "account": func() *AttrDataLoader {
        return NewAttrDataLoader(AttrDataLoaderInits{
            "id": func() *DataLoader {
                return NewDataLoader(100, 1*time.Millisecond,
                    func(keys []Key) (values []Value, errors []error) {
                        // load data using ids...
                    })
            },
            "email": func() *DataLoader {
                return NewDataLoader(100, 1*time.Millisecond,
                    func(keys []Key) (values []Value, errors []error) {
                        // load data using emails...
                    })
            },
        }, ValuePropagators{
            "id": func(v Value, l *AttrDataLoader) {
                // prime the cache with key v.Email = v
            },
            "email": func(v Value, l *AttrDataLoader) {
                // prime the cache with key v.ID = v
            },
        }, )
    },
    "payment": func() *AttrDataLoader {
        return NewAttrDataLoader(AttrDataLoaderInits{
            "id": func() *DataLoader {
                return NewDataLoader(100, 1*time.Millisecond,
                    func(keys []Key) (values []Value, errors []error) {
                        // load data using ids...
                    })
            },
            "date": func() *DataLoader {
                return NewDataLoader(100, 1*time.Millisecond,
                    func(keys []Key) (values []Value, errors []error) {
                        // load data using emails...
                    })
            },
        }, ValuePropagators{
            "id": func(v Value, l *AttrDataLoader) {
                // prime the cache with key v.date = v
            },
            "date": func(v Value, l *AttrDataLoader) {
                // prime the cache with key v.ID = v
            },
        }, )
    },
})
```

### Loading data

Use the following functions which each DataLoader type implements.

* *.Load()*
* *.LoadAll()*
* *.Clear()*
* *.Prime()*

## Meta

Robin Brämer – [@robinbraemer](https://github.com/robinbraemer)

Distributed under the MIT license. See ``LICENSE`` for more information.

[https://github.com/yourname/github-link](https://github.com/dbader/)

The DataLoader is inspired by https://github.com/facebook/dataloaders
and the underlying loader by https://github.com/vektah/dataloaden.

<!-- Markdown link & img dfn's -->
[sourcegraph]: https://sourcegraph.com/github.com/robinbraemer/dataloaders
[sourcegraph-img]: https://sourcegraph.com/github.com/robinbraemer/dataloaders/-/badge.svg

[godoc]: https://godoc.org/github.com/robinbraemer/dataloaders
[godoc-img]: https://godoc.org/github.com/robinbraemer/dataloaders?status.svg