$("title").text("Buckets (倉庫清單) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Buckets (倉庫清單)")
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

const BucketList = cc("div");

function BucketItem(bucket) {
  let cardStyle = "card mb-4";
  let cardHeaderStyle = "card-header";
  let cardBodyStyle = "card-body";

  if (bucket.encrypted) {
    cardStyle += " text-bg-dark";
    cardBodyStyle += "";
    cardHeaderStyle += "";
  } else {
    cardStyle += " border-success";
    cardBodyStyle += " text-success";
    cardHeaderStyle += " text-success";
  }

  return cc("div", {
    id: "B-" + bucket.name,
    classes: cardStyle,
    children: [
      m("div").addClass(cardHeaderStyle).text(bucket.name),
      m("div")
        .addClass(cardBodyStyle)
        .append(
          m("div")
            .addClass("BucketItemBodyRowOne")
            .append(
              m("div").addClass("fw-bold").text(bucket.title),
              m("div").addClass("text-muted").text(bucket.subtitle)
            ),
          m("div")
            .addClass("text-end")
            .append(
              MJBS.createLinkElem("edit-bucket.html?id=" + bucket.id, {
                text: "info",
              })
            )
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
    m(BucketList).addClass("my-5")
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
      } else {
        PageAlert.insert(
          "warning",
          "沒有倉庫, 請返回首頁, 點擊 Create Bucket 新建倉庫."
        );
      }
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}
