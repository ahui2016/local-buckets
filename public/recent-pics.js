$("title").text("Recent pics (最近圖片) - Local Buckets");

const SearchInput = MJBS.createInput("search", "required");
const SearchBtn = MJBS.createButton("search", "primary", "submit");
const SearchInputGroup = cc("form", {
  classes: "input-group",
  children: [
    m(SearchInput).attr({accesskey: "s"}),
    m(SearchBtn).on("click", event => {
      event.preventDefault();
      const pattern = SearchInput.val();
      if (!pattern) {
	MJBS.focus(SearchInput);
	return;
      }
      PageAlert.insert("info", `正在尋找 ${pattern} ...`);
      MJBS.disable(SearchBtn);
      axiosPost({
	url: "/api/search-pics",
	body: {text: pattern},
	alert: PageAlert,
	onSuccess: resp => {
	  const files = resp.data;
	  if (files && files.length > 0) {
	    PageAlert.clear().insert("success", `找到 ${files.length} 個檔案`);
	    FileList.elem().html('');
	    MJBS.appendToList(FileList, files.map(FileItem));
	  } else {
	    PageAlert.insert("warning", "未找到任何檔案");
	  }
	},
	onAlways: () => {
	  MJBS.focus(SearchInput);
	  MJBS.enable(SearchBtn);
	}
      });
    })
  ]
});

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
	m('div').addClass("ShowSearchBtnArea").css({display: "inline"}).append(
	  " | ",
	  MJBS.createLinkElem("#", { text: "Search"})
	    .addClass("ShowSearchBtn")
	    .on("click", event => {
	      event.preventDefault();
	      MJBS.disable(".ShowSearchBtn");
	      $(".ShowSearchBtnArea").fadeOut(2000);
	      SearchInputGroup.show();
	      MJBS.focus(SearchInput);
	    })
	)
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
    navBar.addClass("mt-3 mb-5"),
    m(PageLoading).addClass("my-5"),
    m(SearchInputGroup).addClass("my-3").hide(),
    m(PageAlert).addClass("my-3"),
    m(FileList).addClass("my-3"),
    m(FileEditCanvas),
    bottomDot
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
  initProjectInfo();
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

function initProjectInfo() {
  axiosGet({
    url: "/api/project-status",
    alert: PageAlert,
    onSuccess: (resp) => {
      PageConfig.projectInfo = resp.data;
      initBackupProject(PageConfig.projectInfo, PageAlert)
    },
  });
}
