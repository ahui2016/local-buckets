$("title").text("Recent pics (最近圖片) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Home" }),
        span(" .. Recent pics (最近圖片)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Files" }).addClass("FilesBtn"),
        " | ",
        MJBS.createLinkElem("/buckets.html", { text: "Buckets" }),
        " | ",
        MJBS.createLinkElem("#", { text: "help" }).addClass("HelpBtn")
      )
  );

const PageConfig = {};

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const FileList = cc("div", { classes: "d-flex flex-wrap p-2" });

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

async function init() {
  let bucketID = getUrlParam("bucket");

  PageConfig.bsFileEditCanvas = new bootstrap.Offcanvas(FileEditCanvas.id);

  FileEditCanvas.elem().on("hidden.bs.offcanvas", () => {
    $("#root").css({ marginLeft: "" });
  });
  FileInfoPageCfg.buckets = await getBuckets(PageAlert);
  getWaitingFolder();
  getRecentPics(bucketID);

  initNavButtons(bucketID);
}

function initNavButtons(bucketID) {
  let href = "/recent-files.html";
  if (bucketID) href += `?bucket=${bucketID}`;
  $(".FilesBtn").attr({ href: href });
}

function getRecentPics(bucketID) {
  bucketID = parseInt(bucketID);
  if (bucketID > 0) {
    PageConfig.picsInBucket = true;
  }

  axiosPost({
    url: "/api/recent-pics",
    body: { id: bucketID },
    alert: PageAlert,
    onSuccess: (resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        MJBS.appendToList(FileList, files.map(FileItem));
      } else {
        const errMsg = bucketID
          ? "在本倉庫中未找到任何圖片"
          : "未找到任何圖片, 請返回首頁, 點擊 Upload 上傳圖片.";
        PageAlert.insert("warning", errMsg);
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
