document.addEventListener("DOMContentLoaded", function(e){
    //var modeSwitch = document.getElementById("mode-switch");
    //modeSwitch.addEventListener("click", function(e){
    //    // add cookie
    //    var mode = getCookie("visual-mode");
    //    if (mode == LIGHTMODE) {
    //        setCookie("visual-mode", DARKMODE, 365);
    //    } else {
    //        setCookie("visual-mode", LIGHTMODE, 365);
    //    }
    //    // reload page
    //    location.reload();
    //})
//
    //// Look for the currently active page and add the active class to it
    //let urls = document.querySelectorAll(".navbar-links a");
    //let currentUrl = window.location.href;
    //let path = currentUrl.split("#")[0];
    //for (let i = 0; i < urls.length; i++) {
    //    console.log(urls[i].href.split("#")[0], path)
    //    if (urls[i].href.split("#")[0] == path) {
    //        for (let j = 0; j < urls.length; j++) {
    //            urls[j].classList.remove("active");
    //        }
    //        urls[i].classList.add("active");
    //    }
    //}
//
    //// Look for query parameters, if they match an input name; 
    //// set the value of the input to the query parameter
    //let inputs = document.querySelectorAll("input");
    //let textareas = document.querySelectorAll("textarea");
    //inputs = [...inputs, ...textareas];
    //let queryString = window.location.search;
    //let urlParams = new URLSearchParams(queryString);
    //for (let i = 0; i < inputs.length; i++) {
    //    let name = inputs[i].name;
    //    if (urlParams.has(name)) {
    //        inputs[i].value = urlParams.get(name);
    //    }
    //}
    
    let messages = document.querySelectorAll(".removeself")
    let messageAnimation = [
        { opacity: 1, height: "100%"},
        { opacity: 0, height: "0%"}
    ];
    let messageAnimationOptions = {
        duration: 200,
        easing: 'ease-in-out'
    }
    messages.forEach(function(elem){
        elem.addEventListener("click", function(){
            let anim = elem.animate(messageAnimation, messageAnimationOptions)
            anim.onfinish = () => {
                elem.remove()
            }
        })
    })
    setTimeout(function(){
        for (let i = 0; i < messages.length; i++) {
            setTimeout(function(){
                // Gradually decrease the height of the message, such that the other messages go up
                let height = messages[i].offsetHeight;
                let anim = messages[i].animate([
                    { transform: "translateY(0px)", height: height + "px" },
                    { transform: "translateY(calc(-" + height + "px * 1.5))", height: "10px" }
                ], {
                    duration: 200,
                    easing: 'ease-in-out'
                })
                anim.onfinish = () => {
                    messages[i].remove()
                }
            }, 1000 * i)
        }
    }, 4000)
})