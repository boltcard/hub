<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Bolt 12 demo</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous">
</head>

<script type="module">
    var o = window.location,
        a = "wss://" + o.host + "/websocket";
    let e = new WebSocket(a);
    e.addEventListener("open", t => {
        e.send("test websocket send")
    });
    e.onmessage = function(t) {
        let n = t.data,
        s = document.createElement("li");
        s.classList.add("list-group-item");
        s.textContent = n
        document.getElementById("messages").prepend(s)
    };
</script>

<!-- https://getbootstrap.com/docs/5.3/components/list-group/#basic-example 
<ul class="list-group">
    <li class="list-group-item">An item</li>
    <li class="list-group-item">A second item</li>
</ul>
-->

<body>
    <h1>Bolt 12 offer</h1>
    <div>
        <a href="lightning:{{.QrValue}}">
            <img class="img-fluid" alt="Bolt 12 Offer QR code" src="data:image/png;base64,{{.OfferQrPngEncoded}}" />
        </a>
    </div>

    <ul class="list-group" id="messages">
    </ul>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js" integrity="sha384-YvpcrYf0tY3lHB60NNkmXc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz" crossorigin="anonymous"></script>
</body>

</html>
