# IE XMLHttpRequest bug

This project is a proof of concept for an existing IE11 HTTPS XMLHttpRequest responseText length bug.

The project contains basic javascript XHR client code, and a basic HTTPS server designed to fill the XMLHttpRequest channel with large amounts of text data.

## Details

In IE11, when asynchronous HTTPS XMLHttpRequest responseText's length exceeds a threshold (~60MB < X < ~128MB), IE11 will throw the following exception:

```text
  ERROR Error: Not enough storage is available to complete this operation.

  objectError = {
    description: "Error: Not enough storage is available to complete this operation.",
    message: "Error: Not enough storage is available to complete this operation.",
    name: "Error",
    number: -2147024882,
    stack: "Error: Not enough storage is available to complete this operation.",
    Symbol: rxSubscriber
  }
```

This behavior does not confirm to W3C XHR specifications which make no provisions for exceptions in asynchronous XHR requests. Only synchronous XHR requests are permitted to throw exceptions:

[XHR Spec](https://xhr.spec.whatwg.org)

## Affected Frameworks

As a result, RxJS and Angular do not trap or bubble up the exception. The top level app does not get called back with an error, with no error handling, and no indication the XHR request cannot be read, the top level app remains unaware that the connection is no longer readable.

Here is the specific problem line for Angular 6:

[Angular 6 Src](https://github.com/angular/angular/blob/6.0.x/packages/common/http/src/xhr.ts#L274)

```typescript
if (req.responseType === 'text' && !!xhr.responseText) {
+  try {
     progressEvent.partialText = xhr.responseText;
     // Finally, fire the event.
     observer.next(progressEvent);
+  } catch (error) {
+    const res = new HttpErrorResponse({
+      error,
+      status: 413, // use 413 - Request Entity Too Large
+      statusText: 'IE11 Not enough storage is available to complete this operation',
+    });
+    observer.error(res);
+  }
}
```

For non-Angular, the following should allow the browser to recover:

```javascript
xhr.onreadystatechange = function () {
+  try {
     const text = xhr.responseText;
+  } catch (error) {
+    xhr.abort();
+    return;
+  }
}
```

## Reports

- [StackOverflow](https://stackoverflow.com/search?q=responseText+%22Not+enough+storage+is+available+to+complete+this+operation%22)
- [jQuery](https://github.com/jquery/jquery/issues/3499)
