# Distributed Locking TCP Client 

Simple tcp client using Consul session locks to control transmission to TCP receiver.

## Usage

Default settings will connect to local Consul agent on default http api port (8500/tcp) and look for the receiving TCP server on port localhost:9000. 

Consul session lock acquision can be disabled with the --wait flag to demonstrate parrallel transmission.

Clients may use the --id flag to identify their transmission.

TCP receiver address can be changed with the --server flag.

