var find = "\\."
var rep = "-"

var findUrl = /(-\w+)\//

function changeUrl(str) {
    if (str.includes("host.juicyrout:8091")) {
        return str;
    }
    var replacedStr = str.replace(new RegExp(find, 'g'), rep)
    var replacedStr1 = replacedStr.replace(findUrl, "$1.host.juicyrout:8091/")
    return replacedStr1
}

var constantMock = window.fetch;
window.fetch = function (url, config) {
    var args = Array.prototype.slice.call(arguments)
    console.log.apply(console, args)
    arguments[0] = changeUrl(arguments[0])
    return constantMock.apply(this, arguments)
}

var oldXHROpen = window.XMLHttpRequest.prototype.open;
window.XMLHttpRequest.prototype.open = function (method, url, async, user, password) {
    arguments[1] = changeUrl(arguments[1])

    var args = Array.prototype.slice.call(arguments)
    console.log.apply(console, args)
    this.addEventListener('load', function () {
        console.log('load: ' + this.responseText)
    })

    return oldXHROpen.apply(this, arguments)
}
