**Note:** This is Chatter v2. [Chatter v1](https://github.com/davidbalbert/chatter/tree/v1) is hackier, but implements more of OSPF.

# Chatter

Chatter is a work-in-progress implementation of [OSPF v2](https://www.rfc-editor.org/rfc/rfc2328), an IP routing protocol. 

## Requirements

Chatter currently requires macOS. To port to other OSs, the only thing needed is an implementation of `platformMonitor` for monitoring network interface changes (see [interface_monitor.go](/net/netmon/interface_monitor.go) and [interface_monitor_darwin.go](/net/netmon/interface_monitor_darwin.go)).

## Features

- Dynamic service managment
    - Services are restarted on config changes*.
    - Dependencies (e.g. OSPF depends on InterfaceMonitor to hear about changes to network interface state). Services are started in dependency order.
- GRPC-based API.
- Cisco IOS-style CLI interface with autocomplete, integrated help, and support for entering abbreviated commands.
- A good portion of the OSPF interface state machine, but no actual sending and recieving of packets.
- TODO: OSPF features supported.

*File system monitoring for config changes is not implemented yet, but ServiceManager is ready to support it.

## Compared to Chatter v1

[Chatter v1](https://github.com/davidbalbert/chatter/tree/v1) is a hack, but it does a lot more – it can form adjacencies with neighboring routers and exchange link-state databases.

## Build and run

### Running manually

First, copy chatterd.yaml.example to chatterd.yaml, and update the list of interfaces to match some interfaces on your computer.

In one terminal window, build Chatter, and run chatterd:

```
$ make
$ build/chatterd -config chatterd.yaml -socket /tmp/chatterd.sock
Starting chatterd v0.0.1-dev (2871e53) with uid 501
starting service: APIServer
starting service: InterfaceMonitor
starting service: OSPF
interface event: en0 10.0.0.198/24: InterfaceUp
```

In another, run the CLI. 

```
$ build/chatterc  -socket /tmp/chatterd.sock
connected to chatterd v0.0.1-dev (2871e53; dirty)
chatterc# 
```

### Running from within VS Code

If you run the "chatterd + chatterc" launch configuration, VS Code will open two terminals, and run one program in each. If you select one terminal session and then drag the sidebar entry for the other one on top of the main terminal, you will get a split view where you can see both chatterc and chatterd running at the same time.

### A sample CLI session

Here's an example session in the CLI. "<?>" means pressing the ? key on your keyboard.

```
chatterc# <?>
  exit      Exit the CLI
  quit      Exit the CLI
  show      Show running system information
  shutdown  Shutdown chatterd
chatterc# sh<?>
  show      Show running system information
  shutdown  Shutdown chatterd
chatterc# show<?>
  show  Show running system information
chatterc# show <?>
  interfaces  Interface status and configuration
  processes   Show running processes
  version     Show version
chatterc# show ver
v0.0.1-dev (2871e53; dirty)
chatterc# sh proc
Name               Type               
----------------   ----------------   
APIServer          APIServer          
InterfaceMonitor   InterfaceMonitor   
OSPF               OSPF               
chatterc# sh int 
Name      State   MTU     Addresses                                    
anpi0     Up      1500                                                 
anpi1     Up      1500                                                 
anpi2     Up      1500                                                 
ap1       Up      1500    fe80::f02f:4bff:fe0a:4aeb/64                 
awdl0     Up      1500    fe80::8c8a:eeff:fee7:ad02/64                 
bridge0   Up      1500                                                 
en0       Up      1500    10.0.0.198/24                                
                          2600:4041:59af:3801:422:9667:791:90b3/64     
                          2600:4041:59af:3801:b1c5:affc:7f81:d64a/64   
                          fda7:9c58:dcc6:c443:4b8:d993:5fa4:5a3a/64    
                          fe80::1497:39df:a159:8f3d/64                 
en1       Up      1500                                                 
en2       Up      1500                                                 
en3       Up      1500                                                 
en4       Up      1500                                                 
en5       Up      1500                                                 
en6       Up      1500                                                 
en8       Up      1500    fe80::49:1aff:fe6b:13d0/64                   
gif0      Down    1280                                                 
llw0      Up      1500    fe80::8c8a:eeff:fee7:ad02/64                 
lo0       Up      16384   127.0.0.1/8                                  
                          ::1/128                                      
                          fe80::1/64                                   
stf0      Down    1280                                                 
utun0     Up      1500    fe80::a8c4:3ced:ecaa:70ef/64                 
utun1     Up      1380    fe80::d6ef:c456:7531:468/64                  
utun2     Up      2000    fe80::d93a:92bb:dd8f:c8fd/64                 
utun3     Up      1000    fe80::ce81:b1c:bd2c:69e/64                   
utun4     Up      1380    fe80::f29:35ba:4f89:4550/64                  
utun5     Up      1380    fe80::aa00:c219:2777:e71c/64                 
utun6     Up      1380    fe80::ddd6:69f:f29a:ce7f/64                  
utun7     Up      1380    fe80::cb96:58ed:402:1f3f/64                  
chatterc# exit
$
```

## License

Chatter is copyright David Albert and released under the terms of the MIT License. See LICENSE.txt for details.
