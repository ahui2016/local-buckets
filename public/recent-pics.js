$("title").text("Recent pics (最近圖片) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Recent pics (最近圖片)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "help" }).addClass("HelpBtn"),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
      )
  );

const PageConfig = {};

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const FileList = cc("div", { classes: "d-inline-flex p-2" });

function FileItem(file) {
  const fileItemID = "F-" + file.id;
  const thumbID = "#" + fileItemID + " img";

  let notes = file.notes;
  if (file.keywords) notes = `${notes} [${file.keywords}]`;
  if (notes == "") notes = file.name;

  let headerText = `${file.bucket_name}/${notes}`;
  if (file.encrypted) headerText = "🔒" + headerText;

  const self = cc("div", {
    id: fileItemID,
    children: [
      m("img")
        .attr({
          alt: file.name,
          title: headerText,
        })
        .css({ cursor: "pointer" })
        .on("click", (event) => {
          event.preventDefault();
          $("#root").css({ marginLeft: rootMarginLeft });
          PageConfig.bsFileEditCanvas.show();
          initEditFileForm(file.id, thumbID, true);
        }),
    ],
  });

  self.init = () => {
    axios.get(`/thumbs/${file.id}`).then((resp) => {
      $(thumbID).attr({ src: resp.data });
    });
  };

  return self;
}

$("#root")
  .css(RootCssWide)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(FileEditCanvas),
    m(FileList).addClass("my-5"),
    m("div").text(".").addClass("my-5 text-light")
  );

init();

function init() {
  PageConfig.bsFileEditCanvas = new bootstrap.Offcanvas(FileEditCanvas.id);
  FileEditCanvas.elem().on("hidden.bs.offcanvas", () => {
    $("#root").css({ marginLeft: "" });
  });
  getBuckets();
  getWaitingFolder();
  getRecentPics();
}

function getRecentPics() {
  axiosGet({
    url: "/api/recent-pics",
    alert: PageAlert,
    onSuccess: (resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        MJBS.appendToList(FileList, files.map(FileItem));
      } else {
        PageAlert.insert(
          "warning",
          "未找到任何圖片, 請返回首頁, 點擊 Upload 上傳檔案."
        );
      }
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}

function rebuildThumbs(start, end) {
  axiosPost({
    url: "/api/rebuild-thumbs",
    alert: PageAlert,
    body: { start: start, end: end },
    onSuccess: () => {
      console.log("Success!");
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}
