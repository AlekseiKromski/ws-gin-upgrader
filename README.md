# at-socket-server core 🤯
This repository helps developers, who want to write application with really lightful system, that use web sockets

## Links
- [The main idea of project](#main) 🧨
- [features](#features) 🔪
  - [actions / triggers system](#action_trigger) 👽 **[[DEPRECATED]](#deprecated)** 
  - [hook system](#hook) 🪝
- [Deprecated notes](#deprecated) 👆

## Features <a name="features"></a>

### The main idea of project 🧨 **[[DEPRECATED]](#deprecated)** <a name="main"></a>
This application uses a system of actions and triggers.

#### Action <a name="action_trigger"></a>
Accepts an incoming request from a user on a socket connection. It has its own specific standard (interface), which must be implemented.

Interface realization:
- **SetData(data string)** -> param that will provide Data from request
- **Do() string** -> entry point. **Do()** method will be call after getting user request. Always should return something, but not necessery. **Return string will be passed into trigger SetData() method**
- **TrigType() string** -> there should be trigger "id", that will run after action. Could be empty if you don't want to call trigger
- **SetClient(client Client)** -> setting up the client, that send request. You can get ID in system or manipulate with connection

#### Trigger
After finishing work with the action, the trigger will be automatically called and serves to do the final part of the work on the request. It also has its own interface, which must be implemented

Interface realization:
- **Do()** -> entry point. There should be your code, that will run after Action
- **SetData(data string)** -> setting the data, that you return after **Action.Do()** method
- **SetClient(client Client)** -> pass the client, that send request 
- **SetClients(client []Client)** -> pass all clients, that we have in websocket connection

#### Notes
There are situations when we do not need a trigger and we can simply not specify its type, thus only the action will be processed.

### Hooks <a name="hook"></a>
When you create an application, a special channel will be created that will transmit signals that you can listen to. At the moment, the application core has:

| Hook name    | Description                                                       |
|--------------|-------------------------------------------------------------------|
| CLIENT_ADDED | This hook will be triggered, when user will connected into server |

## Deprecated notes <a name="deprecated"></a>
In the future, core will use simple version of handlers. Action and Triggers will deactivated



