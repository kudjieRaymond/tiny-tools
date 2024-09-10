## How to Setup

### Start the servers

Run in 5 separate terminal the command below
`python -m http.server <port>`

eg :
in terminal 1 => `python3 -m http.server 4001`
in terminal 2 => `python3 -m http.server 4002`
in terminal 3 => `python3 -m http.server 4003`

### Start the Load balancer

Start the Load balancer
`go run main.go`

### Test the Load Balancer

In another terminal send Parallel requests to the load balancer
`for i in {1..10} ; do curl -s -i http://localhost:8080 | grep -i "X-Forwarded-Server" ; done`

You can see the Load balancer outputs a different server URL in a round robin fashion
