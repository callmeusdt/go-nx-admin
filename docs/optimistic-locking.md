# Optimistic locking

Built-in mutable resources expose a `version` integer. A successful update
increments it atomically. Clients that edit an existing resource should send
the version returned by the list or detail response with every update or
delete request.

For example:

```json
{
  "version": 3,
  "status": 0
}
```

The server applies the write as a compare-and-swap operation. If another
request has already advanced the resource, the request returns `409 Conflict`
with `code: RESOURCE_VERSION_CONFLICT`. The stale request never overwrites the
newer value. Clients should keep the draft, reload the current resource, and
let the operator decide whether to discard or merge it; stale writes must not
be retried automatically.

## Covered resources

The version is included for users, roles, menus, APIs, and media. Single-item
updates and deletes use the version when it is supplied. Media deletion checks
the version before deleting the stored object, preserving the existing retry
behavior when storage deletion fails.

For migration compatibility, requests from older clients that omit `version`
retain the previous behavior. New clients must send the value; applications
that require strict concurrency protection should upgrade those clients before
enabling destructive workflows.

This contract is resource-oriented and does not prescribe a database-specific
lock column name or a business application's merge UI. Bulk tree,
permission-set, and multi-record operations require an aggregate revision when
an application needs all-or-nothing conflict detection; a single row version
must not be treated as an aggregate revision.
