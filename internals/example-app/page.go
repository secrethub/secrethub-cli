package example_app

const page = `
<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
	<meta name="description" content="">
	<meta name="author" content="">
	<link rel="icon" href="https://secrethub.io/img/favicon/favicon.ico">

	<title>SecretHub Example App</title>

	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.10.2/css/all.min.css" integrity="sha256-zmfNZmXoNWBMemUOo1XUGFfc0ihGGLYdgtJS3KCr/l0=" crossorigin="anonymous" />
	<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css" integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">

	<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/3.4.1/jquery.min.js" integrity="sha256-CSXorXvZcTkaix6Yvo6HppcZGetbYMGWSFlBw8HfCJo=" crossorigin="anonymous"></script>

	<style>
		html,
		body {
			height: 100%;
		}

		body {
			display: -ms-flexbox;
			display: -webkit-box;
			display: flex;
			-ms-flex-align: center;
			-ms-flex-pack: center;
			-webkit-box-align: center;
			align-items: center;
			-webkit-box-pack: center;
			justify-content: center;
			padding-top: 40px;
			padding-bottom: 40px;
			background-color: #f5f5f5;
		}

		.container {
			width: 100%;
			max-width: 400px;
			padding: 15px;
			margin: 0 auto;
		}
	</style>

	<script>
		function processResult(status, icon, color, result){
            $("#status").html(status);
            $("#animation").attr("class", "fas fa-"+icon+" fa-5x");
            $("#animation").css("color", color);
            if(result !== "") {
                $(".result-container").attr("hidden",false);
                $("#result").html(result);
			}
		}

		function processError(message){
            processResult("An error occurred!", "times", "red", message);
		}

		$(function(){
		    $.get("http://127.0.0.1:{{.}}/api", {}, function(res, status, xhr){
		        if(xhr.status === 200) {
                    processResult("Successfully connected to https://demo.secrethub.io/api!", "check", "green", res);
                } else {
					console.log(res, status, xhr);
                    processError(res);
				}
			}).catch(function(res){
                processError(res.responseText);
			});
		});
	</script>
</head>

<body class="text-center">
<div class="container">
	<img class="mb-4" src="https://secrethub.io/img/secrethub-logo-rgb-shield-square.png" alt="SecretHub logo" width="60" height="60">
	<h1 class="h3 mb-3 font-weight-normal">Example App</h1>
	<p id="status">Trying to connect to https://demo.secrethub.io/api...</p>
	<i id="animation" class="fas fa-spinner fa-spin fa-3x"> </i>
	<div class="result-container mt-3" hidden>
		<h4>Result</h4>
		<div class="card" id="result">
		</div>
	</div>
	<p class="mt-5 mb-3 text-muted">Read the <a href="https://secrethub.io/docs/reference/cli/example-app/" target="_blank">documentation</a> for this app</p>
</div>
</body>
</html>
`
