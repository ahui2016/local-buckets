const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Waiting (等待上傳)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Link1" }).addClass("Link1"),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
      )
  );

const PageAlert = MJBS.createAlert();

const WaitingFileList = cc("ul", { classes: "list-group list-group-flush" });

function FileItem(file) {
  return cc("li", {
    id: "F-" + file.checksum,
    classes: "list-group-item",
    children: [
      span(file.size)
        .addClass("text-muted me-2")
        .text(fileSizeToString(file.size)),
      span(file.name),
    ],
  });
}

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(WaitingFileList).addClass("my-5")
  );

init();

function init() {
  getWaitingFolder();
  getWaitingFiles();
}

// https://briian.com/

function getWaitingFiles() {
  axios
    .get("/api/waiting-files")
    .then((resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        MJBS.appendToList(WaitingFileList, files.map(FileItem));
        PageAlert.insert(
          "light",
          "這裡列出的檔案清單僅供參考, 實際上傳檔案以 waiting 資料夾為準.",
          "no-time"
        );
      } else {
        PageAlert.insert("info", "沒有等待上傳的檔案");
        PageAlert.insert(
          "info",
          "請把檔案放到 waiting 資料夾, 然後刷新本頁面."
        );
      }
    })
    .catch((err) => {
      PageAlert.insert("danger", axiosErrToStr(err, defaultData2str));
    });
}

function getWaitingFolder() {
  axiosGet({
    url: "/api/waiting-folder",
    alert: PageAlert,
    onSuccess: (resp) => {
      PageAlert.insert(
        "light",
        `waiting 資料夾 ➡️ ${resp.data.text}`,
        "no-time"
      );
    },
  });
}
