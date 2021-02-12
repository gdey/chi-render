# Responders

This package provides various responders for transforming
objects in to responses to be sent to the client.

The following Responders are provided out of the box.

  * [JSON](json.go)
  * [XML](xml.go)
  * [HTML](html.go)
  * [PlainText](plain_text.go)

To Register a responder use the `SetResponder` method on
a controller.
