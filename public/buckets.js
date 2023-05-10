$("title").text("Buckets (倉庫清單) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Home" }),
        span(" .. Buckets (倉庫清單)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("/files.html", { text: "Files" }),
        " | ",
        MJBS.createLinkElem("/pics.html", { text: "Pics" }),
        " | ",
        MJBS.createLinkElem("/create-bucket.html", { text: "New" }).addClass(
          "HideIfBackup"
        )
      )
  );

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const BucketList = cc("div");

function BucketItem(bucket) {
  const bucketItemID = "B-" + bucket.name;
  const delBtnID = `#${bucketItemID} .DelBtn`;
  const dangerDelBtnID = `#${bucketItemID} .DangerDelBtn`;
  const buttonsID = `#${bucketItemID} .btn`;

  let cardStyle = "card mb-4";
  let cardHeaderStyle = "card-header";
  let cardBodyStyle = "card-body";
  let btnColor;

  if (bucket.encrypted) {
    cardStyle += " text-bg-dark";
    cardBodyStyle += " text-bg-secondary";
    cardHeaderStyle += "";
    btnColor = "btn-secondary";
  } else {
    cardStyle += " border-success";
    cardBodyStyle += " text-success";
    cardHeaderStyle += " text-success";
    btnColor = "btn-light";
  }

  let bucketName = bucket.name;
  if (bucket.encrypted) bucketName = "🔒" + bucketName;

  let filesCount = `${bucket.FilesCount} files`;
  if (bucket.FilesCount <= 1) {
    filesCount = "";
  }

  const ItemAlert = MJBS.createAlert(`${bucketItemID}-alert`);

  return cc("div", {
    id: "B-" + bucket.name,
    classes: cardStyle,
    children: [
      m("div").addClass(cardHeaderStyle).text(bucketName),
      m("div")
        .addClass(cardBodyStyle)
        .append(
          m("div")
            .addClass("BucketItemBodyRowOne row")
            .append(
              m("div")
                .addClass("col-9")
                .append(
                  m("div").addClass("fw-bold").text(bucket.title),
                  m("div").text(bucket.subtitle)
                ),
              m("div")
                .addClass("col-3 text-end")
                .append(
                  m("div").text(filesCount),
                  m("div").text(`(${fileSizeToString(bucket.TotalSize)})`)
                )
            ),
          m("div")
            .addClass("text-end")
            .append(
              MJBS.createLinkElem("files.html?bucket=" + bucket.id, {
                text: "files",
              }).addClass(`btn btn-sm ${btnColor} me-2`),

              MJBS.createLinkElem("pics.html?bucket=" + bucket.id, {
                text: "pics",
              }).addClass(`btn btn-sm ${btnColor} me-2`),

              MJBS.createLinkElem("waiting.html?bucket=" + bucket.name, {
                text: "upload",
              }).addClass(`btn btn-sm ${btnColor} HideIfBackup me-2`),

              MJBS.createLinkElem("edit-bucket.html?id=" + bucket.id, {
                text: "edit",
              }).addClass(`btn btn-sm ${btnColor} HideIfBackup me-2`),

              MJBS.createLinkElem("#", { text: "del" })
                .addClass(`btn btn-sm ${btnColor} DelBtn HideIfBackup`)
                .attr({ title: "delete" })
                .on("click", (event) => {
                  event.preventDefault();
                  MJBS.disable(delBtnID);
                  ItemAlert.insert(
                    "warning",
                    "等待 3 秒, 點擊紅色的 DELETE 按鈕刪除倉庫 (注意, 一旦刪除, 不可恢復!)."
                  );
                  setTimeout(() => {
                    $(delBtnID).hide();
                    $(dangerDelBtnID).show();
                  }, 2000);
                }),
              MJBS.createLinkElem("#", { text: "DELETE" })
                .addClass(`btn btn-sm btn-danger DangerDelBtn HideIfBackup`)
                .hide()
                .on("click", (event) => {
                  event.preventDefault();
                  MJBS.disable(dangerDelBtnID);
                  axiosPost({
                    url: "/api/delete-bucket",
                    alert: ItemAlert,
                    body: { id: bucket.id },
                    onSuccess: () => {
                      $(buttonsID).hide();
                      ItemAlert.clear().insert("success", "該倉庫已被刪除");
                    },
                    onAlways: () => {
                      MJBS.enable(dangerDelBtnID);
                    },
                  });
                })
            ),
          m(ItemAlert)
        ),
    ],
  });
}

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(BucketList).addClass("my-5"),
    bottomDot
  );

init();

function init() {
  getBuckets();
}

function getBuckets() {
  axiosGet({
    url: "/api/auto-get-buckets",
    alert: PageAlert,
    onSuccess: (resp) => {
      const buckets = resp.data;
      if (buckets && buckets.length > 0) {
        MJBS.appendToList(BucketList, buckets.map(BucketItem));
        initProjectInfo();
      } else {
        PageAlert.insert("warning", "沒有倉庫, 請點擊右上角的 New 新建倉庫.");
      }
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
      initBackupProject(resp.data, PageAlert);
    },
  });
}
