# gRPC

The gRPC component is responsible to run a gRPC server. 
It injects unary and stream interceptors for managing the observability part of the server.

The component is implementing the Patron interface and handles also graceful shutdown via the context.

Setting up a gRPC component is done via the Builder (which follows the builder pattern).  
The builder supports passing in `grpc.ServerOption` via the builder method to set the server up.

Check out the examples folder for an example on how to set it up.