var Mgmt = Mgmt || {};

var showHide = function () {
    $(".hidden").hide();
};

var toggleShow = function(elem){
  if ($(elem).hasClass('hidden')) {
      $(elem).removeClass('hidden');
      $(elem).show();
  } else {
      $(elem).addClass('hidden');
      $(elem).hide();
  }
};
Mgmt.initialize = function () {
    console.log("Initialized");
    showHide();
    $(".show-oids").on("click", function (elem) {
        console.log("Clicked");
        console.log(elem);
        var p = $(elem.target);
        toggleShow($(p).closest("tr").find(".oids"));
        toggleShow($(p).closest("tr").find(".oid-hidden"));
    });
};

window.onload = function () {
    Mgmt.initialize();
};
