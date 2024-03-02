## Notation

Notation based on:
- [ArchiMate](https://en.wikipedia.org/wiki/ArchiMate) (/ˈɑːrkɪmeɪt/ AR-ki-mayt; originally from Architecture-Animate), open and independent enterprise architecture modeling language
- [Entity–relationship model](https://en.wikipedia.org/wiki/Entity%E2%80%93relationship_model), describes interrelated things of interest in a specific domain of knowledge

```mermaid
graph TD

  %% Entities =================================

  Infrastructure{{Infrastructure}}:::H
  Database[(Database)]:::H
  Table:::H
  Field([Field]):::H
  Data:::H
  DataField1([Field1]):::H
  DataField2([Field2]):::H

  Guide[\Guide/]:::H

  ProductLine[[Product Line]]:::S
    ProductLine --- Product1[Product 1]:::S
    ProductLine --- Product2[Product 2]:::S

  SoftwareComponents:::G
  subgraph SoftwareComponents[Group of elements]
    SoftwareComponent[Software Component 1]:::S
    SoftwareComponent2[Software Component 2]:::S
  end
  
  SoftwareService([Software Service]):::S  
  
  User["Human actor (e.g. User)"]:::B
  Company{{"Non-human actor (e.g. Company)"}}:::B
  BusinessProcess(Business Process):::B


  %% Relations =================================

  Infrastructure ---|runs| SoftwareComponents

  SoftwareComponent --- |provides| SoftwareService

  Infrastructure --x Database
  Database ---x|has few| Table
  Table ---x|has few| Field

  SoftwareService -.->|generates| Data

  SoftwareService --- |used by| BusinessProcess
  BusinessProcess --- |assigned to| User

  User --- |uses| Guide

  Data -.->|used by| SoftwareComponent2

  Data --- DataField1
  Data --- DataField2

  Product2 --- |used by| Company
  Company --- |has| BusinessProcess

  classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
  classDef B fill:#FFFFB5,color:#333
  classDef S fill:#B5FFFF
  classDef H fill:#C9E7B7
```