> ## Documentation Index
> Fetch the complete documentation index at: https://api-docs.levanta.io/llms.txt
> Use this file to discover all available pages before exploring further.

# Authentication

> Authenticate your requests to Levanta with an API key

<AccordionGroup>
  <Accordion title="1. Create a Levanta Account" defaultOpen>
    Visit [app.levanta.io](https://app.levanta.io/auth/sign-up) to create a Levanta account.
  </Accordion>

  <Accordion title="2. Generate an API Key" defaultOpen>
    Navigate to Settings > API and click "Generate API Key". This will generate a new
    API key that you can use to authenticate your requests.
  </Accordion>

  <Accordion title="3. Authenticate Requests" defaultOpen>
    To authenticate your requests, include the `Authorization` header with the value `Bearer <api_key>`.
    For example:

    ```javascript theme={null}
    const headers = {
        Authorization: 'Bearer <api_key>'
    };
    await fetch('https://app.levanta.io/api/<creator|seller>/<version>/<endpoint>', {
        method: '...',
        headers
    })
    ```
  </Accordion>
</AccordionGroup>


This documentation is built and hosted on [Mintlify](https://mintlify.com), a developer documentation platform.