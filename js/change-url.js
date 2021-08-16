var baseDomain = "host.juicyrout:8091"
var apiURL = "https://api." + baseDomain

function changeUrl(str) {
    var url;
    try {
        url = new URL(str)
    } catch (_) {
        return str
    }
    if (url.host) {
        url.host = toProxy(url.host)
    }
    return url.toString()
}

function toProxy(domain) {
    if (domain.includes(baseDomain)) {
        return domain;
    }
    var result = ''
    for (var i = 0; i < domain.length; i++) {
        var ch = domain.charAt(i)
        switch (ch) {
            case "-":
                result += "--"
                break
            case ".":
                result += "-"
                break
            default:
                result += ch
        }
    }
    return result + "." + baseDomain
}
