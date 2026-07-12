# API Contracts

API contracts, Protocol Buffer definitions, and schema specifications representing Alexandria's Ubiquitous Language.

## Status

Protocol Buffer definitions are active across multiple domains:
- **Collaboration**: board, friction, pulls, ideas, issues
- **Capture**: etchings
- **Intelligence**: memory, reasoning
- **Development**: tools, hooks, sync
- **Workspace**: spoke, governance
- **Foundation**: errors
- **Observability**: audit
- **Operations**: result
- **Registry**: repos, artifacts, brain, skills, documents, services
- **Domain/Model**: contact, role, party, message, common
- **Messaging**: notifications
- **Analytics**: billing

## Generation

Go client code is generated automatically from these contracts using [Buf](https://buf.build) and output to the `go/contracts` module.

To compile proto definitions manually:

```bash
# From the contracts/ directory
buf generate
```

## Structure

```
├── buf.yaml       # Buf configuration (v2)
├── buf.gen.yaml   # Buf generation template directing output to go/contracts
└── proto/         # Raw protocol buffer files (.proto) grouped by domain
```
