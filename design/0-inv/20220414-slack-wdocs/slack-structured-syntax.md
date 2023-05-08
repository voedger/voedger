# Enums

```javascript
enum ChannelKinds (Public, Private)
```

# Workspaces

```javascript
class sys.ACL extends CDoc{}
class sys.LinkedUser extends CDoc{}

class sys.Workspace{

  // CDoc's

  ACL: []sys.ACL
  Users: []sys.LinkedUser
  Devices: []sys.LinkedDevice
  Groups: []sys.Group
  OrdersAsCDoc: []Order
  
}

class Channel extends sys.Workspace {

  // WDoc
  ChannelMessages: []ChannelMessage

  @Required
  kind: ChannelKinds
}
```

# Docs

```javascript
doc ChannelMessage /*extends WDoc*/ {
  message: Message
  replies: []ThreadMessage
  tag(slack, message, something)
}
```

# Structs

```javascript

struct Message {
  text: string
  typed_at: TimeStampMs
  
  @unique(sys.Creator, Kind)
  reactions: []MessageReaction
}

struct MessageReaction {
  kind: ReactionKind
  time: TimeStampMs

  tag(slack, message, something)  
}



```

# Commands

```javascript
func MakeOrder(order Order)

```

# Views

```javascript
view ChannelMessageView (ChannelMessages) {
  text: message.Text
  message_date_time: sys.created
  author: sys.creator.Name
  reactions: select Kind, count(*) from MessageReaction group by Kind
  top_reactors: select last 10 user, insertDatetime from MessageReaction      
  
//  top_reactors_ivv:
//    fields:[user, insert_datetime]
//    top: 10
//    from: MessageReaction
}
```

# Organizations

- Workspaces can be linked to Organizations
- Each workspace can be linked to a few Organizations (Since channel in Slack can be shared)
- One Organization is the Workspace Owner
- Organization - built-in entity????
