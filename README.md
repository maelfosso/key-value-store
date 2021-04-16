# Simple Key Value Store

Simple program running on one machine to handle set of key-value pair

## Built With

- Go

## Getting Started

### Prerequisites

You need to have these softwares and tools installed

- Go
- curl

### Setup

To get a local copy up and running follow these simple example steps.

- Clone the repository `git clone https://github.com/maelfosso/key-value-store`
- Change your current directory `cd key-value-store`
- Install all the dependencies `go get`
- Launch the app `go run .`

## Use it

From you CLI, run these commands to :

- Save a key/value pair: `curl -X POST -d 'vv' http://localhost:8080/key/k`
- Get a value of the key **k**: `curl http://localhost:8080/key/k`
- Delete a key/value pair: `curl -X DELETE http://localhost:8080/key/k`


## Authors

👤 **Mael FOSSO**

- GitHub: [@maelfosso](https://github.com/maelfosso)
- Twitter: [@maelfosso](https://twitter.com/maelfosso)
- LinkedIn: [LinkedIn](https://www.linkedin.com/in/mael-fosso-650b6346/)

## 🤝 Contributing

Contributions, issues, and feature requests are welcome!

Feel free to check the [issues page](issues/).

## Show your support

Give a ⭐️ if you like this project!

## 📝 License

This project is [MIT](lic.url) licensed.
