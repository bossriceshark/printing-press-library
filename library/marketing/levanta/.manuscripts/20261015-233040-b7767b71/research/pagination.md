> ## Documentation Index
> Fetch the complete documentation index at: https://api-docs.levanta.io/llms.txt
> Use this file to discover all available pages before exploring further.

# Pagination

> Paginate through Levanta API results

Levanta uses cursor-based pagination to break a large query result into smaller chunks
that reduces load on our systems. Not all endpoints support pagination, but those that do
will return a `cursor` field in the response. You can use this cursor to fetch the next
page of results. Page size can be set via the `limit` query parameter and cursor can be
set via the `cursor` query parameter. When there are no more results to return the `cursor`
field will be returned as `null`.

## Example

```javascript theme={null}
let cursor = '';
while (cursor !== null) {
    const url = new URL('https://app.levanta.io/api/<creator|seller>/<version>/<endpoint>');
    const search = new URLSearchParams({
        limit: '100',
        cursor,
    });
    url.search = search.toString();
    const response = await fetch(url.toString(), {
        method: '...',
        headers: {
            Authorization: 'Bearer <api_key>',
        },
    });
    const json = await response.json();
    cursor = json.cursor;
}
```


This documentation is built and hosted on [Mintlify](https://mintlify.com), a developer documentation platform.