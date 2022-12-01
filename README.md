# API Runner

A lightweight test runner for testing http APIs. Define test cases as json and execute them against any server (local or over the network).

Written in Go. Built by the [Warrant](https://warrant.dev/) team.

## Usage

### Install

```shell
go get github.com/warrant-dev/apirunner
```

### Define API tests as json

Sample test file:

```json
{
    "ignoredFields": [
        "internalId"
    ],
    "tests": [
        {
            "name": "createUser",
            "request": {
                "method": "POST",
                "url": "/users",
                "body": {
                    "email": "someemail@gmail.com"
                }
            },
            "expectedResponse": {
                "statusCode": 200,
                "body": {
                    "userId": "{{ createUser.userId }}",
                    "email": "someemail@gmail.com"
                }
            }
        },
        {
            "name": "getUserById",
            "request": {
                "method": "GET",
                "url": "/users/{{ createUser.userId }}"
            },
            "expectedResponse": {
                "statusCode": 200,
                "body": {
                    "userId": "{{ createUser.userId }}",
                    "email": "someemail@gmail.com"
                }
            }
        },
        {
            "name": "deleteUser",
            "request": {
                "method": "DELETE",
                "url": "/users/{{ createUser.userId }}"
            },
            "expectedResponse": {
                "statusCode": 200
            }
        }
    ]
}
```

### Execute tests

```go
import (
	"github.com/warrant-dev/apirunner"
)

// Execute all tests in 'mytestfile.json' and print results
func main() {
    runner, err := apirunner.NewRunner(apirunner.Config{
        BaseUrl:       "http://localhost:8000",
        CustomHeaders: nil,
    }, "mytestfile.json")
    if err != nil {
        panic(err)
    }
    runner.Execute()
}
```

## Features

- Supports all HTTP operations (`GET`, `POST`, `PUT`, `DELETE` etc.)
- Deep comparison of json responses (objects and arrays)
- Inject custom headers via config (useful for passing auth tokens)
- `ignoredFields` to ignore specific attributes during comparison (ex. non-deterministic ids, timestamps)
- Memoization of response attributes to support request chaining. For example, this test references an id of a resource created by a previous request:

```json
{
    "name": "updateResourceTest",
    "request": {
        "method": "PUT",
        "url": "/resources/{{ createResourceTest.Id }}",
        "body": {
            "email": "someupdatedemail@gmail.com"
        }
    },
    "expectedResponse": {
        "statusCode": 200,
        "body": {
            "id": "{{ createResourceTest.Id }}",
            "email": "someupdatedemail@gmail.com"
        }
    }
}
```

## Development

PRs welcome! Clone and develop locally:

```shell
git clone git@github.com:warrant-dev/apirunner.git
cd apirunner
go build
```

## About Warrant

[Warrant](https://warrant.dev/) provides APIs and infrastructure for implementing authorization and access control.
