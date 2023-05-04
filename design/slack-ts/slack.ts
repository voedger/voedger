/*****************************************

  Slack workspace
  
*/

workspace Organization{ // CDoc<Organization>, Workspace descirptor
  collections: [Channels]
}

collection Channels {
  channel: ref(Channel)
}

/******************************************

  Channel workspace
  
*/

tag slack("all about slack")

tag message("all about message")

workspace Channel{
  parent: ref(Organization)
  collections: [MessageReaction, ChannelMessage, ThreadMessage]
}

struct Message {
  text: string
  typed: TimeStampMs
}

collection MessageReaction {
  message: ref (ChannelMessage | ThreadMessage)
  kind: ReactionKind
  time: TimeStampMs
  unique(Message, sys.Creator, Kind)
  tag(slack, message, something)
  sort()
}  
  
collection ChannelMessage {
  message: Message
  tag(slack, message, something)
}

collection ThreadMessage {
  message: Message
  chanel_message: ref(ChannelMessage)
}

view ChannelMessageView (ChannelMessage) {
  text: message.Text
  message_date_time: sys.created
  author: sys.creator.Name
  reactions: select Kind, count from MessageReaction group by Kind
  top_reactors: select last 10 user, insertDatetime from MessageReaction      
}

It is possible to enrich results using references (e.g. ref. to ChannelMessage is known) (some query language is needed)

Collection ops: 
- Read first/last n
- Neighborhood of given ID
- Entire collection/workspace (indexing/import)

Key structure: ((id_hi, id_lo), idx)
Value structure:
- record_data
- next_id_hi // valid for idx=20 (bucket size)
- next_id_lo // valid for idx=20 (bucket size)

qname is not needed :)

