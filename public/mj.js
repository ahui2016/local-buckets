// 受 Mithril 启发的基于 jQuery 实现的极简框架 https://github.com/ahui2016/mj.js

// 安全提醒：使用 mjElement.append() 时，如果直接 append 字符串可以注入 html,
// 建议用 span() 来包裹字符串。

// 这些 class 只是为了方便生成文档，不实际使用。
class mjElement {};
class mjComponent {};
class ComponentOptions {};

/**
 * 函数名 m 来源于 Mithril.js, 也可以理解为 make 的简称，用来创建一个元素。
 * mjElement 就是 `JQuery<HTMLElement>`
 * @param {string | mjComponent} obj
 * @returns {mjElement}
 */
function m(obj) {
  if (typeof obj === "string") {
    return $(document.createElement(obj));
  }
  return obj.view;
}

/**
 * @param {string} name
 * @param {string} id
 * @returns {mjComponent}
 */
function newComponent(name, id) {
  const raw_id = id;
  id = "#" + id;
  return {
    id: id,
    raw_id: raw_id,
    view: m(name).attr("id", raw_id),
    elem: () => $(id),
    find: (selector) => $(id).find(selector),
    hide: () => {
      $(id).hide();
    },
    show: () => {
      $(id).show();
    },
    init: () => {},
  };
}

/**
 * 函数名 cc 意思是 create a component, 用来创建一个简单的组件。
 * component.id 由随机数生成，有可能发生冲突，可根据需要换一种生成 id 的算法。
 * @param {string} name
 * @param {ComponentOptions?} options `{id, text, children, classes, css, attr, prop}`
 * @returns {mjComponent}
 */
function cc(name, options) {
  let id = `r${Math.round(Math.random() * 100000000)}`;

  // 如果没有 options
  if (!options) {
    return newComponent(name, id);
  }

  // 如果有 options
  // 如果有 id 就用指定 id, 否则保留随机 id
  if (options.id) id = options.id;

  const component = newComponent(name, id);

  if (options.attr) component.view.attr(options.attr);
  if (options.prop) component.view.prop(options.prop);
  if (options.css) component.view.css(options.css);
  if (options.classes) component.view.addClass(options.classes);

  if (options.text) {
    component.view.text(options.text);
  } else if (options.children) {
    component.view.append(options.children);
  }
  return component;
}

/**
 * 使用 mjElement.append() 时，如果直接 append 字符串可以注入 html,
 * 建议用 span() 来包裹字符串。
 * @param {string} text
 * @returns {mjElement}
 */
span = function (text) {
  return m("span").text(text);
};
