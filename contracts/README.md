# API Contracts

API contracts, Protocol Buffer definitions, and schema specifications representing Alexandria's Ubiquitous Language.

## Status

Protocol Buffer definitions are active across multiple domains (34 packages):
- **Alx**: email, postmark
- **Analytics**: billing
- **Capture**: etchings
- **Collaboration**: board, friction, ideas, issues, pulls
- **Common**: privacy
- **Deployment**: status
- **Development**: hooks, sync, tools
- **Domain/Model**: common, contact, message, party, role
- **Foundation**: errors
- **Intelligence**: memory, reasoning
- **Messaging**: notifications
- **Observability**: audit
- **Operations**: result
- **Registry**: artifacts, brain, documents, repos, services, skills
- **Timeline**: event
- **Workspace**: governance, spoke

## Versioning

Package versions signal API stability:

- **`v1`** — proven contracts with a real consumer; breaking changes are
  rejected by `buf breaking` in CI. Today this is the five `domain/*`
  packages (`common`, `contact`, `message`, `party`, `role`).
- **`v1alpha1`** — everything else: the shape is published but not yet
  validated by a consumer, and may change. Packages are promoted to `v1`
  deliberately, one at a time, once a consumer proves the shape.

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
