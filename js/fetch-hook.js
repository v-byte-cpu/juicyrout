
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
        console.log("xhrOpenHook", args)
        return oldXHROpen.apply(this, arguments)
    }

    var oldXHRSend = window.XMLHttpRequest.prototype.send
    window.XMLHttpRequest.prototype.send = function () {
        console.log("xhrSendHook", arguments)
        this.withCredentials = true;
        return oldXHRSend.apply(this, arguments)
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
                case "img":
                    console.log("appendChild", arguments)
                    console.log("appendChild", "insert img tag!")
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

    Object.defineProperty(document, "cookie", {
        get() {
            console.log("document.cookie get")
            let req = new XMLHttpRequest();
            req.open("GET", apiURL + "/cookies", false)
            req.withCredentials = true
            req.send()
            console.log("document.cookie get: ", req.responseText)
            return req.responseText
        },
        set(value) {
            console.log("document.cookie set: ", value)
            fetch(apiURL + "/cookies", {
                method: "POST",
                credentials: "include",
                body: value
            }).then(response => {
                console.log("document.cookie set status: ", response.status)
            })
        }
    })
}

// TODO insertBefore, after, before, append, prepend
// TODO insertAdjacentElement, insertAdjacentHTML, insertAdjacentText
