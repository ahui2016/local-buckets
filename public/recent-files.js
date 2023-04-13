$("title").text("Recent files (最近檔案) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Recent files (最近檔案)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Link1" }).addClass("Link1"),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
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

  const ItemAlert = MJBS.createAlert();

  const bodyRowOne = m("div")
    .addClass("mb-2 FileItemBodyRowOne")
    .append(m("div").addClass("text-right FileItemBadges"));

  const bodyRowTwoLeft = m("div")
    .addClass("col-2 text-start")
    .append(span("❤").hide());
  const bodyRowTwoRight = m("div")
    .addClass("col-10 text-end")
    .append(
      span(`(${fileSizeToString(file.size)})`).addClass("me-1"),
      span(file.utime.substr(0, 10))
        .attr({ title: file.utime })
        .addClass("me-1"),
      MJBS.createLinkElem("#", { text: "down" })
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
      MJBS.createLinkElem("#", { text: "view", blank: true })
        .addClass("FileInfoBtn FilePreviewBtn me-1")
        .attr({ title: "preview" })
        .hide(),
      // MJBS.createLinkElem("edit-file.html?id=" + file.id, { text: "info" })
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
        .addClass("FileInfoBtn FileInfoDelBtn me-1")
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

  let headerText = `${file.bucket_name}/${file.name}`;
  if (file.encrypted) headerText = "🔒" + headerText;

  const self = cc("div", {
    id: fileItemID,
    classes: "card mb-4",
    children: [
      m("div").addClass("card-header").text(headerText),
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
    const badges = self.find(".FileItemBadges");
    const rowOne = self.find(".FileItemBodyRowOne");
    if (file.damaged) {
      badges.append(span("DAMAGED").addClass("badge text-bg-danger"));
    }
    if (file.deleted) {
      badges.append(span("DELETED").addClass("badge text-bg-secondary ms-2"));
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
      const previewBtn = self.find(".FilePreviewBtn");
      previewBtn.show();
      if (file.type == "text/md") {
        const css = PageConfig.projectInfo.markdown_style;
        previewBtn.attr({ href: `/md.html?id=${file.id}&css=${css}` });
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
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(FileEditCanvas),
    m(FileList).addClass("my-5"),
    m("div").text(".").addClass("my-5 text-light")
  );

init();

async function init() {
  PageConfig.bsFileEditCanvas = new bootstrap.Offcanvas(FileEditCanvas.id);

  FileEditCanvas.elem().on("hidden.bs.offcanvas", () => {
    $("#root").css({ marginLeft: "" });
  });

  getBuckets();
  getWaitingFolder();
  PageConfig.projectInfo = await getProjectInfo();
  getRecentFiles();
}

function getRecentFiles() {
  axiosGet({
    url: "/api/recent-files",
    alert: PageAlert,
    onSuccess: (resp) => {
      const files = resp.data;
      if (files && files.length > 0) {
        MJBS.appendToList(FileList, files.map(FileItem));
        $(".FileInfoBtn").addClass("btn btn-sm btn-light text-muted");
        $(".FileInfoDangerDelBtn")
          .removeClass("btn-light text-muted")
          .addClass("btn-danger");
      } else {
        PageAlert.insert(
          "warning",
          "未找到任何檔案, 請返回首頁, 點擊 Upload 上傳檔案."
        );
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
      url: "/api/project-info",
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
    fileType.startsWith("text") ||
    fileType.endsWith("pdf")
  );
}
