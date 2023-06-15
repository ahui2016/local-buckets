$("title").text("Files (檔案清單) - Local Buckets");

const BucketID = getUrlParam("bucket");
const BucketName = getUrlParam("bucketname");
const SortBy = getUrlParam("sort");

const SearchInput = MJBS.createInput("search", "required");
const SearchBtn = MJBS.createButton("search", "primary", "submit");
const SearchInputGroup = cc("form", {
  classes: "input-group",
  children: [
    m(SearchInput).attr({ accesskey: "s" }),
    m(SearchBtn).on("click", (event) => {
      event.preventDefault();
      const pattern = SearchInput.val();
      if (!pattern) {
        MJBS.focus(SearchInput);
        return;
      }
      PageAlert.insert("info", `正在尋找 ${pattern} ...`);
      MJBS.disable(SearchBtn);
      axiosPost({
        url: "/api/search-files",
        body: { text: pattern },
        alert: PageAlert,
        onSuccess: (resp) => {
          const files = resp.data;
          if (files && files.length > 0) {
            PageAlert.clear().insert("success", `找到 ${files.length} 個檔案`);
            FileList.elem().html("");
            MJBS.appendToList(FileList, files.map(FileItem));
          } else {
            PageAlert.insert("warning", "未找到任何檔案");
          }
        },
        onAlways: () => {
          MJBS.focus(SearchInput);
          MJBS.enable(SearchBtn);
        },
      });
    }),
  ],
});

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Home" }),
        span(" .. Files (檔案清單)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Pics" }).addClass("PicsBtn"),
        " | ",
        MJBS.createLinkElem("/buckets.html", { text: "Buckets" }),
        m("div")
          .addClass("ShowSearchBtnArea")
          .css({ display: "inline" })
          .append(
            " | ",
            MJBS.createLinkElem("#", { text: "Search" })
              .addClass("ShowSearchBtn")
              .on("click", (event) => {
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

const FileList = cc("div");

function FileItem(file) {
  const fileItemID = "F-" + file.id;
  const fileInfoButtons = `#${fileItemID} .FileInfoBtn`;
  const delBtnID = `#${fileItemID} .FileInfoDelBtn`;
  const dangerDelBtnID = `#${fileItemID} .FileInfoDangerDelBtn`;

  const ItemAlert = MJBS.createAlert(`${fileItemID}-alert`);

  const bodyRowOne = m("div").addClass("mb-2 FileItemBodyRowOne");

  const bodyRowTwoLeft = m("div").addClass("col-2 text-start FileItemLike");

  const bodyRowTwoRight = m("div")
    .addClass("col-10 text-end")
    .append(
      span(`(${fileSizeToString(file.size)})`).addClass("me-1"),
      span(file.utime.substr(0, 10))
        .attr({ title: file.utime })
        .addClass("me-1"),

      MJBS.createLinkElem("#", { text: "DL" })
        .addClass("FileInfoBtn FileInfoDownloadBtn me-1")
        .attr({ title: "download" })
        .on("click", (event) => {
          event.preventDefault();
          event.currentTarget.style.pointerEvents = "none";
          axiosPost({
            url: "/api/download-file",
            alert: ItemAlert,
            body: { id: file.id },
            onSuccess: () => {
              ItemAlert.insert(
                "success",
                `成功下載到 waiting 資料夾 ${PageConfig.waitingFolder}`
              );
            },
            onAlways: () => {
              event.currentTarget.style.pointerEvents = "auto";
            },
          });
        }),

      MJBS.createLinkElem("#", { text: "small" })
        .addClass("FileInfoBtn FileInfoSamllBtn me-1")
        .attr({ title: "下載小圖" })
        .hide()
        .on("click", (event) => {
          event.preventDefault();
          event.currentTarget.style.pointerEvents = "none";
          axiosPost({
            url: "/api/download-small-pic",
            alert: ItemAlert,
            body: { id: file.id },
            onSuccess: () => {
              ItemAlert.insert(
                "success",
                `成功下載到 waiting 資料夾 ${PageConfig.waitingFolder}`
              );
            },
            onAlways: () => {
              event.currentTarget.style.pointerEvents = "auto";
            },
          });
        }),

      MJBS.createLinkElem("#", { text: "view", blank: true })
        .addClass("FileInfoBtn FilePreviewBtn me-1")
        .attr({ title: "preview" })
        .hide(),

      MJBS.createLinkElem("#", { text: "info" })
        .addClass("FileInfoBtn FileInfoEditBtn me-1")
        .on("click", (event) => {
          event.preventDefault();
          $("#root").css({ marginLeft: rootMarginLeft });
          PageConfig.bsFileEditCanvas.show();
          initEditFileForm(
            file.id,
            "#" + fileItemID + " .FileInfoEditBtn",
            false
          );
        }),

      MJBS.createLinkElem("#", { text: "del" })
        .addClass("FileInfoBtn FileInfoDelBtn HideIfBackup me-1")
        .attr({ title: "delete" })
        .on("click", (event) => {
          event.preventDefault();
          PageConfig.bsFileEditCanvas.hide();
          MJBS.disable(delBtnID);
          ItemAlert.insert(
            "warning",
            "等待 3 秒, 點擊紅色的 DELETE 按鈕刪除檔案 (注意, 一旦刪除, 不可恢復!)."
          );
          setTimeout(() => {
            $(delBtnID).hide();
            $(dangerDelBtnID).show();
          }, 2000);
        }),

      MJBS.createLinkElem("#", { text: "DELETE" })
        .addClass("FileInfoBtn FileInfoDangerDelBtn")
        .hide()
        .on("click", (event) => {
          event.preventDefault();
          MJBS.disable(fileInfoButtons);
          axiosPost({
            url: "/api/delete-file",
            alert: ItemAlert,
            body: { id: file.id },
            onSuccess: () => {
              $(fileInfoButtons).hide();
              ItemAlert.clear().insert("success", "該檔案已被刪除");
            },
            onAlways: () => {
              MJBS.enable(fileInfoButtons);
            },
          });
        })
    );

  let bucketName = file.bucket_name;
  if (file.encrypted) bucketName = "🔒" + bucketName;
  const bucketLink = MJBS.createLinkElem("?bucketname=" + file.bucket_name, {
    text: bucketName + "/",
  }).addClass("link-dark fw-bold text-decoration-none");
  const cardHeader = m("div").append(
    span("DAMAGED").addClass("badge text-bg-danger DamagedBadge me-1").hide(),
    bucketLink,
    span(file.name)
  );

  const self = cc("div", {
    id: fileItemID,
    classes: "card mb-4",
    children: [
      m("div").addClass("card-header").append(cardHeader),
      m("div")
        .addClass("card-body")
        .append(
          m("div").append(bodyRowOne),
          m("div").addClass("row").append(bodyRowTwoLeft, bodyRowTwoRight),
          m(ItemAlert)
        ),
    ],
  });

  self.init = () => {
    const rowOne = self.find(".FileItemBodyRowOne");
    const damagedBadge = self.find(".DamagedBadge");
    const fileItemLike = self.find(".FileItemLike");

    self.find(".FileInfoBtn").addClass("btn btn-sm btn-light text-muted");

    self
      .find(".FileInfoDangerDelBtn")
      .removeClass("btn-light text-muted")
      .addClass("btn-danger");

    if (file.like == 1) {
      fileItemLike.text("❤");
    }
    if (file.like > 1) {
      fileItemLike.text(`❤${file.like}`);
    }
    if (file.damaged) {
      damagedBadge.show();
    }
    if (file.notes) {
      rowOne.append(m("div").append(span(file.notes).addClass("text-muted")));
    }
    if (file.keywords) {
      rowOne.append(
        m("div").append(span(`[${file.keywords}]`).addClass("text-muted"))
      );
    }

    if (canBePreviewed(file.type)) {
      const css = PageConfig.projectInfo.markdown_style;
      const previewBtn = self.find(".FilePreviewBtn");
      previewBtn.show();
      if (file.type == "text/md") {
        previewBtn.attr({ href: `/md.html?id=${file.id}&css=${css}` });
      } else if (file.type == "text/plain") {
        previewBtn.attr({ href: `/txt.html?id=${file.id}&css=${css}` });
      } else {
        previewBtn.attr({ href: "/file/" + file.id });
      }
    }
  };

  return self;
}

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("mt-3 mb-5"),
    m(PageLoading).addClass("my-5"),
    m(CurrentBucketAlert).addClass("my-3").hide(),
    m(SearchInputGroup).addClass("my-3").hide(),
    m(PageAlert).addClass("my-3"),
    m(FileList).addClass("my-3"),
    m(MoreBtnArea).addClass("my-5").hide(),
    m(FileEditCanvas),
    bottomDot
  );

init();

async function init() {
  const searchPattern = getUrlParam("search");

  PageConfig.bsFileEditCanvas = new bootstrap.Offcanvas(FileEditCanvas.id);

  FileEditCanvas.elem().on("hidden.bs.offcanvas", () => {
    $("#root").css({ marginLeft: "" });
  });

  initNavButtons(BucketID, BucketName);
  getWaitingFolder();
  FileInfoPageCfg.buckets = await getBuckets(PageAlert);
  PageConfig.projectInfo = await getProjectInfo();

  if (getUrlParam("damaged")) {
    getDamagedFiles();
  } else if (searchPattern) {
    PageLoading.hide();
    MJBS.disable(".ShowSearchBtn");
    $(".ShowSearchBtnArea").hide();
    SearchInputGroup.show();
    SearchInput.setVal(searchPattern);
    SearchBtn.elem().trigger("click");
  } else {
    getFilesLimit(BucketID, BucketName);
  }

  BucketSelect.elem().attr({ accesskey: "s" });
  NotesInput.elem().attr({ accesskey: "n" });
  KeywordsInput.elem().attr({ accesskey: "k" });
  SubmitBtn.elem().attr({ accesskey: "e" });
}

function initNavButtons(bucketID, bucketName) {
  let href = "/pics.html";
  if (bucketID) {
    href += `?bucket=${bucketID}`;
  } else if (bucketName) {
    href += `?bucketname=${bucketName}`;
  }
  $(".PicsBtn").attr({ href: href });
}

function getFilesLimit(bucketID, bucketName) {
  axiosPost({
    url: "/api/files",
    body: { id: parseInt(bucketID), name: bucketName, sort: SortBy, utime: "" },
    alert: PageAlert,
    onSuccess: (resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        const lastUTime = files[files.length - 1].utime.substr(0, 19);
        MoreBtnArea.show();
        MoreFilesDateInput.setVal(lastUTime);
        MJBS.appendToList(FileList, files.map(FileItem));
        initBackupProject(PageConfig.projectInfo, PageAlert);
      } else {
        const errMsg = bucketID
          ? "在本倉庫中未找到任何檔案"
          : "未找到任何檔案, 請返回首頁, 點擊 Upload 上傳檔案.";
        PageAlert.insert("warning", errMsg);
      }
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}

function getMoreFiles() {
  MJBS.disable(MoreFilesForm);
  axiosPost({
    url: "/api/files",
    body: {
      id: parseInt(BucketID),
      name: BucketName,
      sort: SortBy,
      utime: MoreFilesDateInput.val(),
    },
    alert: MoreBtnAlert,
    onSuccess: (resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        const lastUTime = files[files.length - 1].utime.substr(0, 19);
        MoreFilesDateInput.setVal(lastUTime);
        MJBS.appendToList(FileList, files.map(FileItem));
      } else {
        MoreBtnAlert.insert("warning", "沒有更多檔案了.");
        MoreFilesForm.hide();
      }
    },
    onAlways: () => {
      MJBS.enable(MoreFilesForm);
    },
  });
}

function getDamagedFiles() {
  PageAlert.insert("info", "正在瀏覽受損檔案 (damaged files)");
  axiosGet({
    url: "/api/damaged-files",
    alert: PageAlert,
    onSuccess: (resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        MJBS.appendToList(FileList, files.map(FileItem));
        initBackupProject(PageConfig.projectInfo, PageAlert);
      } else {
        PageAlert.insert("warning", "未找到受損檔案");
      }
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}

function getProjectInfo() {
  return new Promise((resolve) => {
    axiosGet({
      url: "/api/project-status",
      alert: PageAlert,
      onSuccess: (resp) => {
        resolve(resp.data);
      },
    });
  });
}

function canBePreviewed(fileType) {
  return (
    fileType.startsWith("image") ||
    fileType.startsWith("video") ||
    fileType.startsWith("text") ||
    fileType.endsWith("pdf")
  );
}

function showMoreButtons() {
  $(".FileInfoSamllBtn").show();
}
