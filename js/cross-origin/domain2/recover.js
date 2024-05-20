var field0 = document.getElementById("input0");
if (sessionStorage.getItem("autosave")) {
    // recover the value of input0 after papge refresh
    field0.value = sessionStorage.getItem("autosave");
}

field0.addEventListener("change", function() {
    sessionStorage.setItem("autosave", field0.value);
});