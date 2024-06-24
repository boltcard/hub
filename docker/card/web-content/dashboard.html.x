<!DOCTYPE html>
<html>

<head>
    <title>dashboard</title>
    <meta http-equiv="refresh" content="10">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet"
        integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous">
</head>

<body>
    <div class="container">
        <h1>Admin dashboard</h1>

        <canvas id="qr"></canvas>

        <p>{{.QrValue}}</p>
        <p>
            Fee credit = {{.FeeCreditSat}} sats<br/>
            Balance = {{.BalanceSat}} sats
        </p>
        <p>
            Updated at {{.UpdateTime}}
        </p>

        <form action="/admin/" method="post">
            <input type="hidden" name="action" value="logout">
            <input class="btn btn-primary mb-3t" type="submit" value="Logout">
        </form>

        <script src="https://cdnjs.cloudflare.com/ajax/libs/qrious/4.0.2/qrious.min.js"></script>

        <script>
            (function () {
                var qr = new QRious({
                    element: document.getElementById('qr'),
                    size: 400,
                    value: '{{.QrValue}}'
                });
            })();
        </script>

        <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js"
            integrity="sha384-YvpcrYf0tY3lHB60NNkmXc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz"
            crossorigin="anonymous"></script>
</body>

</html>