var baseDomain = "host.juicyrout:8091"

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

if (typeof firstRun == 'undefined') {
    var firstRun = false

    var oldFetch = window.fetch
    window.fetch = function (url, config) {
        var args = Array.from(arguments)
        console.log("fetchHook", args)
        args[0] = changeUrl(args[0])
        if (args.length == 1) {
            args.push({ credentials: "include" })
        }
        args[1].credentials = "include"
        return oldFetch.apply(this, args)
    }

    var oldXHROpen = window.XMLHttpRequest.prototype.open
    window.XMLHttpRequest.prototype.open = function (method, url, async, user, password) {
        arguments[1] = changeUrl(arguments[1])

        var args = Array.from(arguments)
        console.log("xhrHook", args)
        this.withCredentials = true;
        return oldXHROpen.apply(this, arguments)
    }

    var oldAppendChild = Node.prototype.appendChild
    Node.prototype.appendChild = function () {
        var child = arguments[0]
        if (child && child.tagName && child.tagName.toLowerCase) {
            var tagName = child.tagName.toLowerCase()
            switch (tagName) {
                case "script":
                    console.log("appendChild", arguments)
                    console.log("appendChild", "insert script tag!")
                    child.removeAttribute("crossorigin")
                    child.src = changeUrl(child.src)
                    break
                case "link":
                    console.log("appendChild", arguments)
                    console.log("appendChild", "insert link tag!")
                    child.removeAttribute("crossorigin")
                    child.href = changeUrl(child.href)
                    break
            }
        }
        return oldAppendChild.apply(this, arguments)
    }
}

// TODO insertBefore, after, before, append, prepend
// TODO insertAdjacentElement, insertAdjacentHTML, insertAdjacentText
