$("title").text("Recent (最近檔案) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Recent (最近檔案)")
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
const PageLoading = MJBS.createLoading(null, "large");

const FileList = cc("div");

function FileItem(file) {
  const bodyRowOne = m("div")
    .addClass("mb-2 FileItemBodyRowOne")
    .append(m("div").addClass("text-right FileItemBadges"));

  const bodyRowTwoLeft = m("div")
    .addClass("col text-start")
    .append(span("❤").hide());
  const bodyRowTwoRight = m("div")
    .addClass("col text-end")
    .append(
      span(`(${fileSizeToString(file.size)})`).addClass("me-2"),
      span(file.utime.substr(0, 10))
        .attr({ title: file.utime })
        .addClass("me-2"),
      MJBS.createLinkElem("edit-file.html?id=" + file.id, { text: "info" })
    );

  let headerText = `${file.bucket_name}/${file.name}`;
  if (file.encrypted) headerText = "🔒" + headerText;

  const self = cc("div", {
    id: "F-" + file.id,
    classes: "card mb-4",
    children: [
      m("div").addClass("card-header").text(headerText),
      m("div")
        .addClass("card-body")
        .append(
          m("div").append(bodyRowOne),
          m("div").addClass("row").append(bodyRowTwoLeft, bodyRowTwoRight)
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
      rowOne.append(
        m("div").append(
          span("Notes: "),
          span(file.notes).addClass("text-muted")
        )
      );
    }
    if (file.keywords) {
      rowOne.append(
        m("div").append(
          span("Keywords: "),
          span(file.keywords).addClass("text-muted")
        )
      );
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
    m(FileList).addClass("my-5")
  );

init();

function init() {
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
