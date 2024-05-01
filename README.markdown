**Note:** This is Chatter v1. [Chatter v2](https://github.com/davidbalbert/chatter) is less hacky, but implements much less of OSPF.

# Chatter

Chatter is a work-in-progress implementation of [OSPF v2](https://www.rfc-editor.org/rfc/rfc2328), an IP routing protocol. 

## Features

- Sends and listens for Hello packets.
- Forms adjacencies with neighbors and exchanges LSAs.

## Build and run

First, edit main.go to set your router ID, and add interfaces and networks. Make sure to configure interfaces as point-to-multipoint. Other types are not well supported.

Then run chatter:

```
$ go run .
Starting ospfd with uid 501
```

## Compared to Chatter v2

[Chatter v2](https://github.com/davidbalbert/chatter) is less hacky, but implements much less of OSPF. It's also structured much better. Chatter v1's use of goroutines is not well thought out.


## License

Chatter is copyright David Albert and released under the terms of the MIT License. See LICENSE.txt for details.

