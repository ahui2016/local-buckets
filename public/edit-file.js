$("title").text("Edit File Attributes (修改檔案屬性) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. 修改檔案屬性")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Link1" }).addClass("Link1"),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
      )
  );

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(FileInfoPageAlert).addClass("my-5"),
    m(FileInfoPageLoading).addClass("my-5"),
    m(EditFileForm).addClass("my-5").hide()
  );

init();

async function init() {
  let fileID = getUrlParam("id");
  if (!fileID) {
    FileInfoPageLoading.hide();
    FileInfoPageAlert.insert("danger", "id is null");
    return;
  }
  fileID = parseInt(fileID);

  await getBucketsPromise();
  initEditFileForm(fileID, null);
}

function getBucketsPromise() {
  return new Promise((resolve) => {
    axiosGet({
      url: "/api/auto-get-buckets",
      alert: MoveToBucketAlert,
      onSuccess: (resp) => {
        FileInfoPageCfg.buckets = resp.data;
        resolve();
      },
    });
  });
}
