$("title").text("Waiting (等待上傳) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Home" }),
        span(" .. Waiting (等待上傳)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "NewNote" }).addClass("NewNoteBtn"),
        " | ",
        MJBS.createLinkElem("/buckets.html", { text: "Buckets" }),
        " | ",
        MJBS.createLinkElem("/files.html", { text: "Files" }),
      )
  );

const PageConfig = {};

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const BucketSelect = cc("select", {
  classes: "form-select",
  children: [
    m("option")
      .prop("selected", true)
      .attr({ value: "" })
      .text("請選擇一個倉庫..."),
  ],
});

const BucketSelectGroup = cc("div", {
  classes: "input-group input-group-lg",
  children: [span("Bucket").addClass("input-group-text"), m(BucketSelect)],
});

function BucketItem(bucket) {
  let text = bucket.title;
  if (bucket.encrypted) text = "🔒" + text;
  return cc("option", {
    id: "B-" + bucket.name,
    attr: { value: bucket.name, title: bucket.name },
    text: text,
  });
}

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
const ImportButton = MJBS.createButton("Import");
const ImportAlert = MJBS.createAlert();
const ImportButtonArea = cc("div", {
  classes: "text-center",
  children: [
    m(ImportAlert),
    m(ImportButton).on("click", (event) => {
      event.preventDefault();
      const bucket_name = BucketSelect.elem().val();
      if (!bucket_name) {
        ImportAlert.insert("warning", "請選擇一個倉庫");
        return;
      }
      MJBS.disable(ImportButton); // --------------------- disable
      axiosPost({
        url: "/api/import-files",
        body: { text: bucket_name },
        alert: ImportAlert,
        onSuccess: () => {
          ImportButton.hide();
          ImportAlert.clear().insert("success", "上傳成功");
          ImportAlert.insert(
            "info",
            "可能仍有待上傳檔案, 3 秒後本頁將自動刷新."
          );
          setTimeout(() => {
            window.location.reload();
          }, 5000);
        },
        onAlways: () => {
          MJBS.enable(ImportButton); // ------------------- enable
        },
      });
    }),
  ],
});

const UploadButton = MJBS.createButton("Upload");
const UploadAlert = MJBS.createAlert();
const UploadButtonArea = cc("div", {
  classes: "text-center",
  children: [
    m(UploadAlert),
    m(UploadButton).on("click", (event) => {
      event.preventDefault();
      const bucket_name = BucketSelect.elem().val();
      if (!bucket_name) {
        UploadAlert.insert("warning", "請選擇一個倉庫");
        return;
      }
      MJBS.disable(UploadButton); // --------------------- disable
      axiosPost({
        url: "/api/upload-new-files",
        body: { text: bucket_name },
        alert: UploadAlert,
        onSuccess: () => {
          UploadAlert.clear().insert("success", "上傳成功");
          UploadAlert.insert(
            "info",
            "點擊本頁右上角的 Files 按鈕可查看新上傳的檔案."
          );
          UploadButton.hide();
        },
        onAlways: () => {
          MJBS.enable(UploadButton); // ------------------- enable
        },
      });
    }),
  ],
});

$("#root")
  .css(RootCssWide)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(SameNameRadioCard).addClass("my-5").hide(),
    m(BucketSelectGroup).addClass("my-5").hide(),
    m(WaitingFileList).addClass("my-5"),
    m(ImportButtonArea).addClass("my-5").hide(),
    m(UploadButtonArea).addClass("my-5").hide(),
    m(PageLoading).addClass("my-5")
  );

init();

async function init() {
  if ((await initBuckets()) == "fail") {
    return;
  }
  initDefaultBucket();
  getWaitingFolder();
  getImportedFiles();
  initNewNoteBtn();
}

function initBuckets() {
  return new Promise((resolve) => {
    axiosGet({
      url: "/api/auto-get-buckets",
      alert: PageAlert,
      onSuccess: (resp) => {
        const buckets = resp.data;
        if (buckets && buckets.length > 0) {
          PageConfig.buckets = buckets;
          MJBS.appendToList(BucketSelect, buckets.map(BucketItem));
          resolve("success");
        } else {
          PageAlert.insert(
            "warning",
            "沒有倉庫, 請返回首頁, 點擊 Create Bucket 新建倉庫."
          );
          PageLoading.hide();
          resolve("fail");
        }
      },
    });
  });
}

function initDefaultBucket() {
  const bucket = getUrlParam("bucket");
  if (!bucket) return;
  $("#B-" + bucket).prop({ selected: true });
}

function getWaitingFiles() {
  axios
    .get("/api/waiting-files")
    .then((resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        BucketSelectGroup.show();
        UploadButtonArea.show();
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
      getWaitingFilesErrorHandler(err, PageAlert);
    })
    .then(() => {
      PageLoading.hide();
    });
}

function getImportedFiles() {
  axios
    .get("/api/imported-files")
    .then((resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        BucketSelectGroup.show();
        ImportButtonArea.show();
        MJBS.appendToList(WaitingFileList, files.map(FileItem));
        ImportAlert.insert(
          "info",
          "發現可導入(import)的檔案, 如果想當作新檔案上傳, 請進入 waiting 資料夾刪除同名 toml 檔案.",
          "no-time"
        );
        PageAlert.insert(
          "light",
          "這裡列出的檔案清單僅供參考, 實際上傳檔案以 waiting 資料夾為準.",
          "no-time"
        );
      } else {
        getWaitingFiles();
      }
    })
    .catch((err) => {
      getWaitingFilesErrorHandler(err, PageAlert);
    })
    .then(() => {
      PageLoading.hide();
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

function getWaitingFilesErrorHandler(err, alert) {
  if (err.response) {
    const respData = err.response.data;
    if (typeof respData === "string") {
      alert.insert("danger", respData);
      return;
    }
    if (err.response.data.errType == "ErrSameNameFiles") {
      const errSameName = respData;
      console.log(errSameName);
      alert.insert("warning", "檔案名稱重複, 請處理.");
      SameNameRadioCard.show();
      SameNameRadioCard.init(errSameName.file);
      return;
    }
    alert.insert("danger", JSON.stringify(respData));
    return;
  }

  if (err.request) {
    const errMsg =
      err.request.status +
      " The request was made but no response was received.";
    alert.insert("danger", errMsg);
    return;
  }

  alert.insert("danger", err.message);
}

function initNewNoteBtn() {
  $(".NewNoteBtn").on("click", (event) => {
    event.preventDefault();
    axiosGet({
      url: "/api/create-new-note",
      alert: PageAlert,
      onSuccess: (resp) => {
        PageAlert.insert("success", `已生成文字檔案 ${resp.data.text}`);
      },
    });
  });
}
