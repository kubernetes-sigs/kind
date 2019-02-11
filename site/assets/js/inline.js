/* used when sidebar is manually toggled */
function toggleSidebar() {
    if (document.body.classList.contains('sidebar-collapsed')) {
        window.localStorage.setItem('sidebar-collapsed', 'false');
        showSideBar();
    } else {
        window.localStorage.setItem('sidebar-collapsed', 'true');
        hideSideBar();
    }
}

function showSideBar() {
    document.body.classList.remove('sidebar-collapsed');
}

function hideSideBar() {
    document.body.classList.add('sidebar-collapsed');
}

/* get the page width */
function getWidth() {
    return Math.max(
        document.body.scrollWidth,
        document.documentElement.scrollWidth,
        document.body.offsetWidth,
        document.documentElement.offsetWidth,
        document.documentElement.clientWidth
    );
}

var oldWidth;
document.addEventListener("DOMContentLoaded", function () {
    // note: the default state of the page on load is collapsed
    var manualCollapsed = window.localStorage.getItem('sidebar-collapsed');
    var width = getWidth();
    // if we're now under 900px don't unhide it
    // otherwise only unhide if the user has not yet manually toggled it
    // or has toggled it back to visible
    if (width > 900 && (manualCollapsed == 'false' || manualCollapsed == null)) {
        showSideBar();
    }
    // setup listener for width change
    oldWidth = width;
    window.addEventListener("resize", function () {
        // bail out early if it wasn't the width that changed
        var width = getWidth();
        if (oldWidth == width) {
            oldWidth = width;
            return;
        }
        oldWidth = width;
        // if we're now under 900px, hide it
        if (width <= 900) {
            hideSideBar();
        } else {
            // otherwise only unhide if the user has not yet manually toggled it
            // or has toggled it back to visible
            var manualCollapsed = window.localStorage.getItem('sidebar-collapsed');
            if (manualCollapsed == 'false' || manualCollapsed == null) {
                showSideBar();
            }
        }
    });
});
