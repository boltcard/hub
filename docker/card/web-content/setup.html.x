<!DOCTYPE html>
<html>
    <head>
        <title>setup admin</title>

        <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet"
        integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous">    
    </head>
    <body>
        <div class="container">
        <h1>Setup admin password</h1>

        <form action="/admin/" method="post">
            <input type="hidden" name="action" value="setup">
            <label class="col-form-label" for="pw">Password:</label>
            <input class="form-control" type="text" id="pw" name="pw"><br><br>
            <input class="btn btn-primary mb-3t" type="submit" value="Submit">
          </form>

        <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js"
        integrity="sha384-YvpcrYf0tY3lHB60NNkmXc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz"
        crossorigin="anonymous"></script>  
    </body>
</html>
